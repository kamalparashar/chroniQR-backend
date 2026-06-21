-- SQL Schema for chroniQR Backend (Supabase/PostgreSQL)

-- Enable pgcrypto extension for gen_random_uuid() if not enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Clients table (extends Supabase auth.users)
CREATE TABLE IF NOT EXISTS public.clients (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    ga4_measurement_id TEXT,
    ga4_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- QR Codes table
CREATE TABLE IF NOT EXISTS public.qr_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    short_code VARCHAR(50) UNIQUE NOT NULL,
    short_url TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    destination_type VARCHAR(50) NOT NULL,
    destination_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    utm_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    style_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    tags TEXT[] NOT NULL DEFAULT '{}'::text[],
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    ga4_tracking_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- QR Scans table for analytics
CREATE TABLE IF NOT EXISTS public.qr_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    qr_id UUID NOT NULL REFERENCES public.qr_codes(id) ON DELETE CASCADE,
    ip INET,
    user_agent TEXT,
    device_type VARCHAR(50),
    os VARCHAR(50),
    browser VARCHAR(50),
    referrer TEXT,
    language VARCHAR(100),
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    utm_term VARCHAR(255),
    utm_content VARCHAR(255),
    country VARCHAR(255),
    region VARCHAR(255),
    city VARCHAR(255),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    device_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indices for performance
CREATE INDEX IF NOT EXISTS idx_qr_codes_client_id ON public.qr_codes(client_id);
CREATE INDEX IF NOT EXISTS idx_qr_codes_short_code ON public.qr_codes(short_code);
CREATE INDEX IF NOT EXISTS idx_qr_scans_qr_id ON public.qr_scans(qr_id);
CREATE INDEX IF NOT EXISTS idx_qr_scans_created_at ON public.qr_scans(created_at);

-- Trigger to automatically create a profile in public.clients when a user signs up via Supabase Auth
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = ''
AS $$
BEGIN
  INSERT INTO public.clients (id)
  VALUES (new.id);
  RETURN NEW;
END;
$$;

-- Revoke public execute so anon/authenticated roles cannot call it via REST API
-- (It is only invoked internally by the trigger, never by users directly)
REVOKE EXECUTE ON FUNCTION public.handle_new_user() FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION public.handle_new_user() FROM anon;
REVOKE EXECUTE ON FUNCTION public.handle_new_user() FROM authenticated;

CREATE OR REPLACE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

