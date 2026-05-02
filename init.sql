CREATE TABLE companies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    company_id INT NOT NULL,
    email VARCHAR(255) UNIQUE,
    CONSTRAINT fk_company FOREIGN KEY (company_id) REFERENCES companies(id)
);