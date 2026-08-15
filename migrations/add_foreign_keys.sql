ALTER TABLE role_permissions
  ADD CONSTRAINT fk_role_permissions_role
  FOREIGN KEY (role_id) REFERENCES roles(id);

ALTER TABLE customer
  ADD CONSTRAINT fk_customer_area
  FOREIGN KEY (area_id) REFERENCES areas(id);

ALTER TABLE customer
  ADD CONSTRAINT fk_customer_sales_representative
  FOREIGN KEY (sales_representative_id) REFERENCES users(id);

ALTER TABLE customer
  ADD CONSTRAINT fk_customer_company
  FOREIGN KEY (company_id) REFERENCES company(id);

ALTER TABLE customer_installations
  ADD CONSTRAINT fk_customer_installations_customer
  FOREIGN KEY (customer_id) REFERENCES customer(id);

ALTER TABLE customer_installations
  ADD CONSTRAINT fk_customer_installations_technician
  FOREIGN KEY (technician_id) REFERENCES users(id)
  ON DELETE SET NULL ON UPDATE CASCADE;
