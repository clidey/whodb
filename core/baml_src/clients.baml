// Learn more about clients at https://docs.boundaryml.com/docs/snippets/clients/overview
//
// NOTE: These are fallback clients used when dynamic client creation fails.
// Primary client configuration is done at runtime via CreateDynamicBAMLClient()
// using user-selected providers, models, and API keys from the UI.

// Default fallback client - uses OpenAI if environment variable is set
client<llm> DefaultClient {
  provider openai
  retry_policy Exponential
  options {
    model "gpt-4o"
    api_key env.OPENAI_API_KEY
  }
}

// Retry policies for robust API calls
retry_policy Constant {
  max_retries 3
  strategy {
    type constant_delay
    delay_ms 200
  }
}

retry_policy Exponential {
  max_retries 2
  strategy {
    type exponential_backoff
    delay_ms 300
    multiplier 1.5
    max_delay_ms 10000
  }
}
