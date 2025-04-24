-- Add UUID extension if not already available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create packages table with UUID primary key
CREATE TABLE IF NOT EXISTS packages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,  -- Keep name unique but use UUID as PK
    version VARCHAR(100),
    description TEXT,
    author TEXT,
    homepage TEXT,
    repository TEXT,
    license TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    downloads BIGINT,
    popularity_score FLOAT,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS packages_name_idx ON packages(name);
CREATE INDEX IF NOT EXISTS packages_downloads_idx ON packages(downloads DESC);
CREATE INDEX IF NOT EXISTS packages_popularity_idx ON packages(popularity_score DESC);

-- Create package scripts table referencing package UUID
CREATE TABLE IF NOT EXISTS package_scripts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    package_id UUID NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    script_type VARCHAR(50) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (package_id, script_type)  -- Each package can have only one of each script type
);

-- Create indexes for package scripts
CREATE INDEX IF NOT EXISTS package_scripts_package_id_idx ON package_scripts(package_id);
CREATE INDEX IF NOT EXISTS package_scripts_type_idx ON package_scripts(script_type);