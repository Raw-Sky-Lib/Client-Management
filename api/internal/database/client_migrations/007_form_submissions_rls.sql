-- Enable RLS on form_submissions so anon-key access is controlled by policy.
-- Without this, Supabase only grants anon SELECT by default — no INSERT or UPDATE.
ALTER TABLE form_submissions ENABLE ROW LEVEL SECURITY;

-- Client website (anon key) can INSERT new submissions from contact/lead forms.
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'form_submissions'
        AND policyname = 'anon_insert_form_submissions'
    ) THEN
        CREATE POLICY anon_insert_form_submissions
        ON form_submissions FOR INSERT TO anon WITH CHECK (true);
    END IF;
END $$;

-- Portal frontend (anon key) can SELECT all submissions.
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'form_submissions'
        AND policyname = 'anon_select_form_submissions'
    ) THEN
        CREATE POLICY anon_select_form_submissions
        ON form_submissions FOR SELECT TO anon USING (true);
    END IF;
END $$;

-- Portal frontend (anon key) can mark submissions as read.
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'form_submissions'
        AND policyname = 'anon_update_form_submissions'
    ) THEN
        CREATE POLICY anon_update_form_submissions
        ON form_submissions FOR UPDATE TO anon USING (true) WITH CHECK (true);
    END IF;
END $$;
