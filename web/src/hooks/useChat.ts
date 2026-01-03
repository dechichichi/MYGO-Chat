import { useState, useCallback } from 'react';
import { Message, ChatResponse, PhilosopherType } from '../types';

export function useChat() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId] = useState(() => `session_${Date.now()}`);

  const sendMessage = useCallback(async (content: string, philosopher: PhilosopherType) => {
    const userMessage: Message = {
      id: `msg_${Date.now()}`,
      role: 'user',
      content,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setIsLoading(true);

    try {
      const response = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionId,
          message: content,
          philosopher,
        }),
      });

      if (!response.ok) throw new Error('请求失败');

      const data: ChatResponse = await response.json();

      const assistantMessage: Message = {
        id: `msg_${Date.now()}`,
        role: 'assistant',
        content: data.response,
        timestamp: new Date(),
        philosopher: data.philosopher,
      };

      setMessages(prev => [...prev, assistantMessage]);
      return data;
    } catch (error) {
      console.error('Chat error:', error);
      const errorMessage: Message = {
        id: `msg_${Date.now()}`,
        role: 'assistant',
        content: '抱歉，系统暂时出了点问题...迷子でもいい，但现在真的连不上了。',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMessage]);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [sessionId]);

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  return { messages, isLoading, sendMessage, clearMessages };
}
