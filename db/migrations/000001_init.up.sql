CREATE TABLE IF NOT EXISTS llm_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt TEXT NOT NULL,
    response TEXT NOT NULL,
    latency_ms INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);