import os

OLLAMA_BASE_URL: str = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
PORT: int = int(os.getenv("PORT", "8000"))
LOG_LEVEL: str = os.getenv("LOG_LEVEL", "info").upper()

# Phi-3 model name as registered in Ollama
PHI3_MODEL: str = os.getenv("PHI3_MODEL", "phi3:mini")

# Timeout for Ollama inference calls (seconds)
OLLAMA_TIMEOUT: float = float(os.getenv("OLLAMA_TIMEOUT", "10"))
