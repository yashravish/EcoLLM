"""
Rule-based + Phi-3-assisted prompt optimizer.

Rules run first (zero latency). Phi-3 refinement is attempted only when
rule confidence is below threshold; it degrades gracefully on timeout.
"""

import logging
from dataclasses import dataclass, field

import httpx

import config

log = logging.getLogger(__name__)

SYSTEM_REFINE = (
    "You are a prompt engineer. Rewrite the user's prompt to be clearer and more "
    "specific while preserving the original intent. Output ONLY the rewritten prompt "
    "with no explanation, preamble, or markdown."
)


@dataclass
class OptimizationResult:
    original: str
    optimized: str
    rules_applied: list[str] = field(default_factory=list)
    used_phi3: bool = False
    confidence: float = 0.8


def _apply_rules(prompt: str) -> tuple[str, list[str]]:
    """Return (optimized_prompt, list_of_rule_names_applied)."""
    rules_applied: list[str] = []
    p = prompt

    # Truncate excessively long prompts: keep first + last 2000 chars
    if len(p) > 4000:
        p = p[:2000] + "\n...[context truncated]...\n" + p[-2000:]
        rules_applied.append("truncate_excessive_context")

    # Strip trailing whitespace
    stripped = p.rstrip()
    if stripped != p:
        p = stripped
        rules_applied.append("remove_trailing_whitespace")

    # Add language guidance for code-related prompts
    code_keywords = {"code", "function", "debug", "fix", "implement", "write a", "python", "javascript"}
    if any(kw in p.lower() for kw in code_keywords) and "language:" not in p.lower():
        p = p + "\n\nPlease specify the programming language if not already clear."
        rules_applied.append("add_language_guidance_for_code")

    # Add format guidance for explanation prompts
    explain_keywords = {"explain", "describe", "summarize", "what is", "how does"}
    if any(kw in p.lower() for kw in explain_keywords) and "format" not in p.lower():
        p = p + "\n\nProvide a clear, structured explanation."
        rules_applied.append("add_format_guidance_for_explanations")

    return p, rules_applied


def _estimate_confidence(prompt: str, rules_applied: list[str]) -> float:
    confidence = 0.8
    if len(prompt) < 20:
        confidence -= 0.2
    if len(prompt) > 200:
        confidence += 0.1
    if len(rules_applied) > 2:
        confidence -= 0.1
    return max(0.0, min(1.0, confidence))


async def optimize(prompt: str) -> OptimizationResult:
    optimized, rules_applied = _apply_rules(prompt)
    confidence = _estimate_confidence(prompt, rules_applied)

    if confidence < 0.7:
        try:
            refined = await _phi3_refine(optimized)
            return OptimizationResult(
                original=prompt,
                optimized=refined,
                rules_applied=rules_applied,
                used_phi3=True,
                confidence=min(confidence + 0.2, 1.0),
            )
        except Exception as e:
            log.warning("phi3 refinement failed, using rule output: %s", e)

    return OptimizationResult(
        original=prompt,
        optimized=optimized,
        rules_applied=rules_applied,
        used_phi3=False,
        confidence=confidence,
    )


async def _phi3_refine(prompt: str) -> str:
    payload = {
        "model": config.PHI3_MODEL,
        "messages": [
            {"role": "system", "content": SYSTEM_REFINE},
            {"role": "user", "content": prompt},
        ],
        "stream": False,
    }
    async with httpx.AsyncClient(timeout=config.OLLAMA_TIMEOUT) as client:
        resp = await client.post(
            f"{config.OLLAMA_BASE_URL}/api/chat",
            json=payload,
        )
        resp.raise_for_status()
        data = resp.json()
        return data["message"]["content"].strip()
