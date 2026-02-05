package react

// DefaultReActInstruction 默认 ReAct 行为说明（可追加到 system prompt）
const DefaultReActInstruction = `

【ReAct 推理与行动】
请按以下方式与用户交流：
1. **Thought（思考）**：先简短思考当前用户消息的含义、情绪，以及是否需要借助工具（如回忆、查歌词、感知氛围等）再回复。
2. **Action（行动）**：若需要工具，则调用相应工具；若不需要，则直接给出回复（不要调用工具）。
3. **Observation（观察）**：调用工具后，你会收到工具返回的结果，请根据结果继续思考或给出最终回复。

请始终先输出你的思考（Thought），再决定是调用工具还是直接回复。若调用工具，下一轮你会看到 Observation，再据此给出对用户的最终回复。`
