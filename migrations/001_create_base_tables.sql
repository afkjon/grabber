CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY, 
    status TEXT NOT NULL, 
    parameters TEXT NOT NULL, 
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS shops (
    id SERIAL PRIMARY KEY, 
    name TEXT NOT NULL, 
    parameters TEXT NOT NULL, 
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

create table if not exists countries (
    id SERIAL PRIMARY KEY,
    name TEXT
);

CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name TEXT,
    country_id INT,
    CONSTRAINT fk_country
        FOREIGN KEY (country_id)
        REFERENCES countries(id)
        ON DELETE CASCADE
);

create table if not exists cities (
    id SERIAL PRIMARY KEY,
    name TEXT,
    region_id INT,
    CONSTRAINT fk_regions
        FOREIGN KEY (region_id)
        REFERENCES regions(id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS shops (
    id SERIAL PRIMARY KEY, 
    name TEXT NOT NULL,
    city_id INT,
    price TEXT,
    station TEXT,
    station_distance TEXT,
    address TEXT,
    tabelog_url TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cities
        FOREIGN KEY (city_id)
        REFERENCES cities(id)
        ON DELETE SET NULL
);
