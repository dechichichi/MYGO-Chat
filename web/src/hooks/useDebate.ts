import { useState, useCallback, useRef } from 'react';
import { DebateResponse, PhilosopherType } from '../types';

interface DebateConfig {
  topic: string;
  proStance: string;
  conStance: string;
  proPhilosophers: PhilosopherType[];
  conPhilosophers: PhilosopherType[];
}

export function useDebate() {
  const [debate, setDebate] = useState<DebateResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const pollingRef = useRef<number | null>(null);

  const startDebate = useCallback(async (config: DebateConfig) => {
    setIsLoading(true);
    setDebate(null);

    try {
      const response = await fetch('/api/debate/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          topic: config.topic,
          pro_stance: config.proStance,
          con_stance: config.conStance,
          pro_philosophers: config.proPhilosophers,
          con_philosophers: config.conPhilosophers,
          async: true,
        }),
      });

      if (!response.ok) throw new Error('请求失败');

      const data: DebateResponse = await response.json();
      setDebate(data);

      // 开始轮询状态
      if (data.id) {
        pollStatus(data.id);
      }

      return data;
    } catch (error) {
      console.error('Debate error:', error);
      setDebate({ status: 'failed', error: '启动讨论失败' });
      throw error;
    }
  }, []);

  const pollStatus = useCallback(async (debateId: string) => {
    const poll = async () => {
      try {
        const response = await fetch(`/api/debate/status?id=${debateId}`);
        if (!response.ok) throw new Error('查询失败');

        const data: DebateResponse = await response.json();
        setDebate(data);

        if (data.status === 'running' || data.status === 'pending') {
          pollingRef.current = window.setTimeout(poll, 2000);
        } else {
          setIsLoading(false);
        }
      } catch (error) {
        console.error('Poll error:', error);
        setIsLoading(false);
      }
    };

    poll();
  }, []);

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearTimeout(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  return { debate, isLoading, startDebate, stopPolling };
}
