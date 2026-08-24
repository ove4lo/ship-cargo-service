CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('supplier', 'manager');
CREATE TYPE voyage_status AS ENUM ('planned', 'loading', 'departed', 'completed');
CREATE TYPE booking_status AS ENUM ('pending', 'confirmed', 'partial', 'rejected', 'cancelled');
CREATE TYPE booking_priority AS ENUM ('urgent', 'normal', 'low');
CREATE TYPE item_status AS ENUM ('pending', 'placed', 'waitlisted');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'supplier',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vessels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    max_weight_kg DOUBLE PRECISION NOT NULL CHECK (max_weight_kg > 0),
    max_volume_m3 DOUBLE PRECISION NOT NULL CHECK (max_volume_m3 > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE voyages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vessel_id UUID NOT NULL REFERENCES vessels(id),
    route VARCHAR(500) NOT NULL,
    departure_date DATE NOT NULL,
    status voyage_status NOT NULL DEFAULT 'planned',
    reserved_weight_kg DOUBLE PRECISION NOT NULL DEFAULT 0,
    reserved_volume_m3 DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT positive_reserved_weight CHECK (reserved_weight_kg >= 0),
    CONSTRAINT positive_reserved_volume CHECK (reserved_volume_m3 >= 0)
);

CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voyage_id UUID NOT NULL REFERENCES voyages(id),
    user_id UUID NOT NULL REFERENCES users(id),
    priority booking_priority NOT NULL DEFAULT 'normal',
    status booking_status NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE booking_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    description VARCHAR(500) NOT NULL,
    weight_kg DOUBLE PRECISION NOT NULL CHECK (weight_kg > 0),
    volume_m3 DOUBLE PRECISION NOT NULL CHECK (volume_m3 > 0),
    status item_status NOT NULL DEFAULT 'pending'
);

CREATE INDEX idx_voyages_vessel_id ON voyages(vessel_id);
CREATE INDEX idx_voyages_departure_date ON voyages(departure_date);
CREATE INDEX idx_voyages_status ON voyages(status);
CREATE INDEX idx_bookings_voyage_id ON bookings(voyage_id);
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_booking_items_booking_id ON booking_items(booking_id);