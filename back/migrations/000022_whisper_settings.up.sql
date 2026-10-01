CREATE TABLE whisper_settings (
  singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
  active_model text NOT NULL DEFAULT 'tiny',
  preset text NOT NULL DEFAULT 'balanced',
  beam_size integer,
  temperature double precision,
  max_segment_length integer,
  split_on_word boolean,
  initial_prompt text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO whisper_settings(singleton) VALUES (true);
