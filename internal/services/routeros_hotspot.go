package services

import (
	"fmt"
	"net"
	"strings"
)

type HotspotUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Profile     string `json:"profile"`
	Server      string `json:"server"`
	Address     string `json:"address"`
	LimitUptime string `json:"limit_uptime"`
	Comment     string `json:"comment"`
	Disabled    bool   `json:"disabled"`
}

type HotspotUserProfile struct {
	Name             string `json:"name"`
	RateLimit        string `json:"rate_limit"`
	SharedUsers      string `json:"shared_users"`
	SessionTimeout   string `json:"session_timeout"`
	IdleTimeout      string `json:"idle_timeout"`
	KeepaliveTimeout string `json:"keepalive_timeout"`
}

type HotspotActiveSession struct {
	ID       string `json:"id"`
	User     string `json:"user"`
	Address  string `json:"address"`
	MAC      string `json:"mac"`
	Server   string `json:"server"`
	Uptime   string `json:"uptime"`
	BytesIn  string `json:"bytes_in"`
	BytesOut string `json:"bytes_out"`
	LoginBy  string `json:"login_by"`
}

type HotspotUserInput struct {
	Password    *string `json:"password"`
	Profile     *string `json:"profile"`
	Address     *string `json:"address"`
	LimitUptime *string `json:"limit_uptime"`
	Comment     *string `json:"comment"`
	Disabled    *bool   `json:"disabled"`
}

type HotspotDeviceBinding struct {
	Address         string `json:"address"`
	MAC             string `json:"mac"`
	BindingID       string `json:"binding_id"`
	BindingType     string `json:"binding_type"`
	BindingDisabled bool   `json:"binding_disabled"`
	CanManage       bool   `json:"can_manage"`
	CanRestore      bool   `json:"can_restore"`
	Reason          string `json:"reason,omitempty"`
	BindingComment  string `json:"-"`
}

const deviceIsolationMarkerPrefix = "[CRM-DEVICE-ISOLATION original-type="

func (c *RouterOSAPI) ResolveHotspotDeviceBindings(addresses []string) ([]HotspotDeviceBinding, error) {
	uniqueAddresses := make([]string, 0, len(addresses))
	seen := make(map[string]struct{}, len(addresses))
	for _, rawAddress := range addresses {
		ip := net.ParseIP(strings.TrimSpace(rawAddress))
		if ip == nil || ip.To4() == nil {
			if strings.TrimSpace(rawAddress) == "" {
				continue
			}
			return nil, fmt.Errorf("invalid IPv4 device address %q", rawAddress)
		}
		address := ip.To4().String()
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		uniqueAddresses = append(uniqueAddresses, address)
	}
	if len(uniqueAddresses) == 0 {
		return []HotspotDeviceBinding{}, nil
	}

	leases, err := c.Command("/ip/dhcp-server/lease/print")
	if err != nil {
		return nil, fmt.Errorf("failed to read DHCP leases: %w", err)
	}
	arpEntries, err := c.Command("/ip/arp/print")
	if err != nil {
		return nil, fmt.Errorf("failed to read ARP entries: %w", err)
	}
	bindingRecords, err := c.Command("/ip/hotspot/ip-binding/print")
	if err != nil {
		return nil, fmt.Errorf("failed to read Hotspot IP bindings: %w", err)
	}

	results := make([]HotspotDeviceBinding, 0, len(uniqueAddresses))
	for _, address := range uniqueAddresses {
		macs := make(map[string]struct{})
		for _, record := range leases {
			if record["address"] == address && record["mac-address"] != "" {
				macs[strings.ToUpper(record["mac-address"])] = struct{}{}
			}
		}
		for _, record := range arpEntries {
			if record["address"] == address && record["mac-address"] != "" {
				macs[strings.ToUpper(record["mac-address"])] = struct{}{}
			}
		}
		for _, record := range bindingRecords {
			if record["address"] == address && record["mac-address"] != "" {
				macs[strings.ToUpper(record["mac-address"])] = struct{}{}
			}
		}

		resolution := HotspotDeviceBinding{Address: address}
		if len(macs) == 0 {
			resolution.Reason = "MAC address could not be resolved from DHCP, ARP, or IP Binding"
			results = append(results, resolution)
			continue
		}
		if len(macs) != 1 {
			resolution.Reason = "IP address resolves to multiple MAC addresses"
			results = append(results, resolution)
			continue
		}
		for mac := range macs {
			resolution.MAC = mac
		}

		matchingBindings := make([]map[string]string, 0, 1)
		for _, record := range bindingRecords {
			if strings.EqualFold(record["mac-address"], resolution.MAC) && (record["address"] == address || record["address"] == "") {
				matchingBindings = append(matchingBindings, record)
			}
		}
		if len(matchingBindings) == 0 {
			resolution.Reason = "No Hotspot IP Binding exists for this device"
			results = append(results, resolution)
			continue
		}
		if len(matchingBindings) > 1 {
			resolution.Reason = "Multiple Hotspot IP Bindings match this device"
			results = append(results, resolution)
			continue
		}

		binding := hotspotIPBindingFromRecord(matchingBindings[0])
		resolution.BindingID = binding.ID
		resolution.BindingType = binding.Type
		resolution.BindingDisabled = binding.Disabled
		resolution.BindingComment = binding.Comment
		resolution.CanRestore = binding.Type == "blocked" && hasDeviceIsolationMarker(binding.Comment)
		resolution.CanManage = binding.ID != "" &&
			(binding.Type == "regular" || binding.Type == "bypassed" || resolution.CanRestore)
		if binding.ID == "" {
			resolution.Reason = "RouterOS did not return an ID for the matching IP Binding"
		} else if binding.Type == "blocked" && !resolution.CanRestore {
			resolution.Reason = "This binding is already blocked and its previous type was not recorded by CRM"
		} else if !resolution.CanManage {
			resolution.Reason = fmt.Sprintf("Unsupported Hotspot IP Binding type %q", binding.Type)
		}
		results = append(results, resolution)
	}
	return results, nil
}

func (c *RouterOSAPI) SetHotspotDeviceBindingType(address, bindingType string) error {
	bindingType = strings.ToLower(strings.TrimSpace(bindingType))
	if bindingType != "blocked" && bindingType != "restore" {
		return fmt.Errorf("unsupported Hotspot IP binding type %q", bindingType)
	}
	resolutions, err := c.ResolveHotspotDeviceBindings([]string{address})
	if err != nil {
		return err
	}
	if len(resolutions) != 1 {
		return fmt.Errorf("could not resolve Hotspot device at %q", address)
	}
	resolution := resolutions[0]
	if !resolution.CanManage {
		return fmt.Errorf("cannot safely update device at %s: %s", resolution.Address, resolution.Reason)
	}
	targetType := "blocked"
	comment := resolution.BindingComment
	if bindingType == "blocked" {
		if resolution.BindingType == "blocked" {
			return fmt.Errorf("device at %s is already blocked", resolution.Address)
		}
		if resolution.BindingType != "regular" && resolution.BindingType != "bypassed" {
			return fmt.Errorf("cannot safely isolate device at %s because its current binding type is %q", resolution.Address, resolution.BindingType)
		}
		comment = addDeviceIsolationMarker(comment, resolution.BindingType)
		if len(comment) > 255 {
			return fmt.Errorf("cannot safely isolate device at %s because its IP Binding comment would exceed 255 bytes", resolution.Address)
		}
	} else {
		previousType, originalComment, ok := removeDeviceIsolationMarker(comment)
		if resolution.BindingType != "blocked" || !ok {
			return fmt.Errorf("cannot safely restore device at %s because its original binding type is unknown", resolution.Address)
		}
		targetType = previousType
		comment = originalComment
	}
	if _, err := c.Command(
		"/ip/hotspot/ip-binding/set",
		"=.id="+resolution.BindingID,
		"=type="+targetType,
		"=comment="+comment,
	); err != nil {
		return fmt.Errorf("failed to set Hotspot binding for %s (%s): %w", resolution.Address, resolution.MAC, err)
	}
	return nil
}

func addDeviceIsolationMarker(comment, bindingType string) string {
	if _, original, ok := removeDeviceIsolationMarker(comment); ok {
		comment = original
	}
	if comment != "" {
		comment += "\n"
	}
	return comment + deviceIsolationMarkerPrefix + bindingType + "]"
}

func hasDeviceIsolationMarker(comment string) bool {
	_, _, ok := removeDeviceIsolationMarker(comment)
	return ok
}

func removeDeviceIsolationMarker(comment string) (string, string, bool) {
	start := strings.LastIndex(comment, deviceIsolationMarkerPrefix)
	if start < 0 {
		return "", comment, false
	}
	endRelative := strings.IndexByte(comment[start:], ']')
	if endRelative < 0 {
		return "", comment, false
	}
	end := start + endRelative
	previousType := comment[start+len(deviceIsolationMarkerPrefix) : end]
	if previousType != "regular" && previousType != "bypassed" {
		return "", comment, false
	}
	original := comment[:start] + comment[end+1:]
	original = strings.TrimSuffix(original, "\n")
	return previousType, original, true
}

func (c *RouterOSAPI) ListHotspotUsers() ([]HotspotUser, error) {
	records, err := c.Command("/ip/hotspot/user/print")
	if err != nil {
		return nil, fmt.Errorf("failed to list Hotspot users: %w", err)
	}
	users := make([]HotspotUser, 0, len(records))
	for _, record := range records {
		users = append(users, hotspotUserFromRecord(record))
	}
	return users, nil
}

func (c *RouterOSAPI) ListHotspotUserProfiles() ([]HotspotUserProfile, error) {
	records, err := c.Command("/ip/hotspot/user/profile/print")
	if err != nil {
		return nil, fmt.Errorf("failed to list Hotspot user profiles: %w", err)
	}
	profiles := make([]HotspotUserProfile, 0, len(records))
	for _, record := range records {
		profiles = append(profiles, HotspotUserProfile{
			Name:             record["name"],
			RateLimit:        record["rate-limit"],
			SharedUsers:      record["shared-users"],
			SessionTimeout:   record["session-timeout"],
			IdleTimeout:      record["idle-timeout"],
			KeepaliveTimeout: record["keepalive-timeout"],
		})
	}
	return profiles, nil
}

func (c *RouterOSAPI) ListHotspotActiveSessions() ([]HotspotActiveSession, error) {
	records, err := c.Command("/ip/hotspot/active/print")
	if err != nil {
		return nil, fmt.Errorf("failed to list active Hotspot sessions: %w", err)
	}
	sessions := make([]HotspotActiveSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, HotspotActiveSession{
			ID:       record[".id"],
			User:     record["user"],
			Address:  record["address"],
			MAC:      record["mac-address"],
			Server:   record["server"],
			Uptime:   record["uptime"],
			BytesIn:  record["bytes-in"],
			BytesOut: record["bytes-out"],
			LoginBy:  record["login-by"],
		})
	}
	return sessions, nil
}

func (c *RouterOSAPI) CreateHotspotUser(name, password, profile, comment string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return fmt.Errorf("Hotspot username must be between 1 and 64 characters")
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("Hotspot password is required")
	}
	users, err := c.findHotspotUsersByName(name)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return fmt.Errorf("Hotspot user %q already exists", name)
	}

	attributes := []string{"=name=" + name, "=password=" + password}
	if profile = strings.TrimSpace(profile); profile != "" {
		attributes = append(attributes, "=profile="+profile)
	}
	if comment = strings.TrimSpace(comment); comment != "" {
		attributes = append(attributes, "=comment="+comment)
	}
	if _, err := c.Command("/ip/hotspot/user/add", attributes...); err != nil {
		return fmt.Errorf("failed to create Hotspot user %q: %w", name, err)
	}
	return nil
}

func (c *RouterOSAPI) UpdateHotspotUser(name string, input HotspotUserInput) error {
	user, err := c.getHotspotUserByName(name)
	if err != nil {
		return err
	}
	attributes := []string{"=.id=" + user.ID}
	if input.Password != nil {
		if strings.TrimSpace(*input.Password) == "" {
			return fmt.Errorf("Hotspot password cannot be empty")
		}
		attributes = append(attributes, "=password="+*input.Password)
	}
	if input.Profile != nil {
		attributes = append(attributes, "=profile="+strings.TrimSpace(*input.Profile))
	}
	if input.Address != nil {
		attributes = append(attributes, "=address="+strings.TrimSpace(*input.Address))
	}
	if input.LimitUptime != nil {
		attributes = append(attributes, "=limit-uptime="+strings.TrimSpace(*input.LimitUptime))
	}
	if input.Comment != nil {
		attributes = append(attributes, "=comment="+strings.TrimSpace(*input.Comment))
	}
	if input.Disabled != nil {
		attributes = append(attributes, "=disabled="+routerOSBoolean(*input.Disabled))
	}
	if len(attributes) == 1 {
		return fmt.Errorf("no Hotspot user fields to update")
	}
	if _, err := c.Command("/ip/hotspot/user/set", attributes...); err != nil {
		return fmt.Errorf("failed to update Hotspot user %q: %w", name, err)
	}
	return nil
}

func (c *RouterOSAPI) DeleteHotspotUser(name string) error {
	user, err := c.getHotspotUserByName(name)
	if err != nil {
		return err
	}
	if _, err := c.Command("/ip/hotspot/user/remove", "=.id="+user.ID); err != nil {
		return fmt.Errorf("failed to delete Hotspot user %q: %w", name, err)
	}
	return nil
}

func (c *RouterOSAPI) KickHotspotUser(name string) (int, error) {
	users, err := c.findHotspotUsersByName(name)
	if err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, fmt.Errorf("Hotspot user %q not found", name)
	}

	records, err := c.Command("/ip/hotspot/active/print")
	if err != nil {
		return 0, fmt.Errorf("failed to list active Hotspot sessions: %w", err)
	}
	kicked := 0
	for _, record := range records {
		if record["user"] != name || record[".id"] == "" {
			continue
		}
		if _, err := c.Command("/ip/hotspot/active/remove", "=.id="+record[".id"]); err != nil {
			return kicked, fmt.Errorf("failed to disconnect Hotspot session for %q: %w", name, err)
		}
		kicked++
	}
	return kicked, nil
}

func (c *RouterOSAPI) getHotspotUserByName(name string) (HotspotUser, error) {
	users, err := c.findHotspotUsersByName(name)
	if err != nil {
		return HotspotUser{}, err
	}
	if len(users) == 0 {
		return HotspotUser{}, fmt.Errorf("Hotspot user %q not found", name)
	}
	return users[0], nil
}

func (c *RouterOSAPI) findHotspotUsersByName(name string) ([]HotspotUser, error) {
	records, err := c.Command("/ip/hotspot/user/print")
	if err != nil {
		return nil, fmt.Errorf("failed to read Hotspot users: %w", err)
	}
	users := make([]HotspotUser, 0, 1)
	for _, record := range records {
		if record["name"] == name && record[".id"] != "" {
			users = append(users, hotspotUserFromRecord(record))
		}
	}
	return users, nil
}

func hotspotUserFromRecord(record map[string]string) HotspotUser {
	return HotspotUser{
		ID:          record[".id"],
		Name:        record["name"],
		Profile:     record["profile"],
		Server:      record["server"],
		Address:     record["address"],
		LimitUptime: record["limit-uptime"],
		Comment:     record["comment"],
		Disabled:    record["disabled"] == "true" || record["disabled"] == "yes",
	}
}

func routerOSBoolean(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func (c *RouterOSAPI) ListHotspotIPBindings() ([]HotspotIPBinding, error) {
	records, err := c.Command("/ip/hotspot/ip-binding/print")
	if err != nil {
		return nil, fmt.Errorf("failed to list Hotspot IP bindings: %w", err)
	}

	bindings := make([]HotspotIPBinding, 0, len(records))
	for _, record := range records {
		bindings = append(bindings, hotspotIPBindingFromRecord(record))
	}
	return bindings, nil
}

func (c *RouterOSAPI) GetHotspotIPBindingByMAC(macAddress string) (*HotspotIPBinding, error) {
	mac, err := normalizeHotspotMAC(macAddress)
	if err != nil {
		return nil, err
	}

	bindings, err := c.findHotspotIPBindingsByMAC(mac)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, fmt.Errorf("no Hotspot IP binding found for MAC address %s", mac)
	}
	return &bindings[0], nil
}

func (c *RouterOSAPI) SetHotspotIPBindingType(macAddress, bindingType string) error {
	bindingType = strings.ToLower(strings.TrimSpace(bindingType))
	if bindingType != "regular" && bindingType != "bypassed" {
		return fmt.Errorf("unsupported Hotspot IP binding type %q", bindingType)
	}
	return c.updateHotspotIPBindings(macAddress, "=type="+bindingType)
}

func (c *RouterOSAPI) SetHotspotIPBindingDisabled(macAddress string, disabled bool) error {
	value := "no"
	if disabled {
		value = "yes"
	}
	return c.updateHotspotIPBindings(macAddress, "=disabled="+value)
}

func (c *RouterOSAPI) findHotspotIPBindingsByMAC(macAddress string) ([]HotspotIPBinding, error) {
	records, err := c.Command("/ip/hotspot/ip-binding/print")
	if err != nil {
		return nil, fmt.Errorf("failed to read Hotspot IP bindings: %w", err)
	}

	bindings := make([]HotspotIPBinding, 0, 1)
	for _, record := range records {
		binding := hotspotIPBindingFromRecord(record)
		if binding.ID != "" && strings.EqualFold(binding.MAC, macAddress) {
			bindings = append(bindings, binding)
		}
	}
	return bindings, nil
}

func (c *RouterOSAPI) updateHotspotIPBindings(macAddress, attribute string) error {
	mac, err := normalizeHotspotMAC(macAddress)
	if err != nil {
		return err
	}

	bindings, err := c.findHotspotIPBindingsByMAC(mac)
	if err != nil {
		return err
	}
	if len(bindings) == 0 {
		return fmt.Errorf("no Hotspot IP binding found for MAC address %s", mac)
	}

	for _, binding := range bindings {
		if _, err := c.Command(
			"/ip/hotspot/ip-binding/set",
			"=.id="+binding.ID,
			attribute,
		); err != nil {
			return fmt.Errorf("failed to update Hotspot IP binding %s: %w", binding.ID, err)
		}
	}
	return nil
}

func normalizeHotspotMAC(value string) (string, error) {
	mac, err := net.ParseMAC(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid MAC address %q", value)
	}
	return strings.ToUpper(mac.String()), nil
}

func hotspotIPBindingFromRecord(record map[string]string) HotspotIPBinding {
	return HotspotIPBinding{
		ID:       record[".id"],
		MAC:      record["mac-address"],
		Address:  record["address"],
		Type:     record["type"],
		Server:   record["server"],
		Comment:  record["comment"],
		Disabled: record["disabled"] == "true" || record["disabled"] == "yes",
	}
}
