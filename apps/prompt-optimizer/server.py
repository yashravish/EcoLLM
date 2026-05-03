"""
EcoLLM Prompt Optimizer — FastAPI server.

Exposes POST /optimize consumed by the Go API's phi3_client.go.
"""

import logging
import sys

import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

import config
import optimizer

logging.basicConfig(
    stream=sys.stdout,
    level=getattr(logging, config.LOG_LEVEL, logging.INFO),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
log = logging.getLogger(__name__)

app = FastAPI(title="EcoLLM Prompt Optimizer", version="0.1.0")


class OptimizeRequest(BaseModel):
    prompt: str


class OptimizeResponse(BaseModel):
    optimized_prompt: str
    rules_applied: list[str]
    used_phi3: bool
    confidence: float


@app.get("/health")
async def health() -> dict:
    return {"status": "ok", "service": "prompt-optimizer"}


@app.post("/optimize", response_model=OptimizeResponse)
async def optimize_endpoint(req: OptimizeRequest) -> OptimizeResponse:
    if not req.prompt or not req.prompt.strip():
        raise HTTPException(status_code=400, detail="prompt must not be empty")

    result = await optimizer.optimize(req.prompt)
    return OptimizeResponse(
        optimized_prompt=result.optimized,
        rules_applied=result.rules_applied,
        used_phi3=result.used_phi3,
        confidence=result.confidence,
    )


if __name__ == "__main__":
    uvicorn.run("server:app", host="0.0.0.0", port=config.PORT, log_level=config.LOG_LEVEL.lower())
