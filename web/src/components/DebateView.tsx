import { useState } from 'react';
import { PhilosopherType, PHILOSOPHERS, DebateRecord } from '../types';
import { useDebate } from '../hooks/useDebate';

const PRESET_TOPICS = [
  {
    topic: '乐队对我们来说意味着什么？',
    proStance: '乐队是我们表达自我、寻找归属的地方',
    conStance: '乐队让我们学会了面对困难和成长',
  },
  {
    topic: '迷茫的时候应该怎么办？',
    proStance: '迷茫时应该停下来倾听内心的声音',
    conStance: '迷茫时应该继续前进，在行动中找到方向',
  },
  {
    topic: '友情和梦想哪个更重要？',
    proStance: '友情是支撑我们追逐梦想的力量',
    conStance: '梦想是让友情更有意义的目标',
  },
];

export function DebateView() {
  const { debate, isLoading, startDebate } = useDebate();
  const [topic, setTopic] = useState(PRESET_TOPICS[0].topic);
  const [proStance, setProStance] = useState(PRESET_TOPICS[0].proStance);
  const [conStance, setConStance] = useState(PRESET_TOPICS[0].conStance);
  const [proTeam, setProTeam] = useState<PhilosopherType[]>(['tomori', 'anon']);
  const [conTeam, setConTeam] = useState<PhilosopherType[]>(['taki', 'soyo']);

  const handlePresetSelect = (index: number) => {
    const preset = PRESET_TOPICS[index];
    setTopic(preset.topic);
    setProStance(preset.proStance);
    setConStance(preset.conStance);
  };

  const toggleTeamMember = (team: 'pro' | 'con', member: PhilosopherType) => {
    if (team === 'pro') {
      if (proTeam.includes(member)) {
        if (proTeam.length > 1) setProTeam(proTeam.filter(m => m !== member));
      } else {
        setProTeam([...proTeam, member]);
        setConTeam(conTeam.filter(m => m !== member));
      }
    } else {
      if (conTeam.includes(member)) {
        if (conTeam.length > 1) setConTeam(conTeam.filter(m => m !== member));
      } else {
        setConTeam([...conTeam, member]);
        setProTeam(proTeam.filter(m => m !== member));
      }
    }
  };

  const handleStart = () => {
    startDebate({
      topic,
      proStance,
      conStance,
      proPhilosophers: proTeam,
      conPhilosophers: conTeam,
    });
  };

  const getPhaseLabel = (phase: string) => {
    const labels: Record<string, string> = {
      opening: '开场发言',
      questioning: '质询交锋',
      free_debate: '自由辩论',
      closing: '总结陈词',
    };
    return labels[phase] || phase;
  };

  const getPhaseColor = (phase: string) => {
    const colors: Record<string, string> = {
      opening: 'bg-emerald-500',
      questioning: 'bg-amber-500',
      free_debate: 'bg-blue-500',
      closing: 'bg-pink-500',
    };
    return colors[phase] || 'bg-gray-500';
  };

  return (
    <div className="flex flex-col h-full">
      {/* 配置面板 */}
      {!debate && (
        <div className="p-6 space-y-6 overflow-y-auto">
          <div>
            <h2 className="text-2xl font-bold mb-2">🎸 乐队讨论会</h2>
            <p className="text-white/60">选择话题和参与成员，开始一场讨论</p>
          </div>

          {/* 预设话题 */}
          <div>
            <label className="block text-sm font-medium mb-2">预设话题</label>
            <div className="flex gap-2 flex-wrap">
              {PRESET_TOPICS.map((preset, i) => (
                <button
                  key={i}
                  onClick={() => handlePresetSelect(i)}
                  className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                    topic === preset.topic
                      ? 'bg-pink-500 text-white'
                      : 'bg-white/10 hover:bg-white/20'
                  }`}
                >
                  {preset.topic}
                </button>
              ))}
            </div>
          </div>

          {/* 话题输入 */}
          <div>
            <label className="block text-sm font-medium mb-2">讨论话题</label>
            <input
              type="text"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              className="w-full bg-white/10 border border-white/20 rounded-xl px-4 py-3 focus:outline-none focus:border-pink-500"
            />
          </div>

          {/* 立场设置 */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-2 text-emerald-400">正方立场</label>
              <textarea
                value={proStance}
                onChange={(e) => setProStance(e.target.value)}
                rows={2}
                className="w-full bg-emerald-500/10 border border-emerald-500/30 rounded-xl px-4 py-3 focus:outline-none focus:border-emerald-500 resize-none"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2 text-rose-400">反方立场</label>
              <textarea
                value={conStance}
                onChange={(e) => setConStance(e.target.value)}
                rows={2}
                className="w-full bg-rose-500/10 border border-rose-500/30 rounded-xl px-4 py-3 focus:outline-none focus:border-rose-500 resize-none"
              />
            </div>
          </div>

          {/* 队伍选择 */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-2 text-emerald-400">正方成员</label>
              <div className="space-y-2">
                {PHILOSOPHERS.map((char) => (
                  <button
                    key={char.type}
                    onClick={() => toggleTeamMember('pro', char.type)}
                    className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg transition-colors ${
                      proTeam.includes(char.type)
                        ? 'bg-emerald-500/30 border border-emerald-500'
                        : 'bg-white/5 border border-white/10 hover:bg-white/10'
                    }`}
                  >
                    <span>{char.avatar}</span>
                    <span>{char.name}</span>
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium mb-2 text-rose-400">反方成员</label>
              <div className="space-y-2">
                {PHILOSOPHERS.map((char) => (
                  <button
                    key={char.type}
                    onClick={() => toggleTeamMember('con', char.type)}
                    className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg transition-colors ${
                      conTeam.includes(char.type)
                        ? 'bg-rose-500/30 border border-rose-500'
                        : 'bg-white/5 border border-white/10 hover:bg-white/10'
                    }`}
                  >
                    <span>{char.avatar}</span>
                    <span>{char.name}</span>
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* 开始按钮 */}
          <button
            onClick={handleStart}
            disabled={isLoading || proTeam.length === 0 || conTeam.length === 0}
            className="w-full py-4 bg-gradient-to-r from-pink-500 to-rose-500 rounded-xl font-bold text-lg hover:opacity-90 disabled:opacity-50 transition-all"
          >
            开始讨论
          </button>
        </div>
      )}

      {/* 讨论进行中/结果 */}
      {debate && (
        <div className="flex flex-col h-full">
          {/* 状态栏 */}
          <div className="p-4 border-b border-white/10 bg-black/20">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-bold">{debate.topic}</h3>
                <div className="flex items-center gap-2 mt-1">
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    debate.status === 'running' ? 'bg-amber-500' :
                    debate.status === 'completed' ? 'bg-emerald-500' :
                    debate.status === 'failed' ? 'bg-red-500' : 'bg-gray-500'
                  }`}>
                    {debate.status === 'running' ? '进行中' :
                     debate.status === 'completed' ? '已完成' :
                     debate.status === 'failed' ? '失败' : '等待中'}
                  </span>
                  {debate.current_phase && (
                    <span className="text-sm text-white/60">
                      当前阶段: {getPhaseLabel(debate.current_phase)}
                    </span>
                  )}
                </div>
              </div>
              <button
                onClick={() => window.location.reload()}
                className="px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg transition-colors"
              >
                新讨论
              </button>
            </div>
          </div>

          {/* 讨论记录 */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {debate.records?.map((record, index) => (
              <div key={index} className={`debate-record ${record.phase}`}>
                <div className="flex items-center gap-2 mb-2">
                  <span className={`px-2 py-0.5 rounded text-xs ${getPhaseColor(record.phase)}`}>
                    {getPhaseLabel(record.phase)}
                  </span>
                  <span className="font-bold">{record.speaker_name}</span>
                </div>
                <p className="text-white/80 whitespace-pre-wrap leading-relaxed">
                  {record.content}
                </p>
              </div>
            ))}

            {isLoading && debate.status === 'running' && (
              <div className="flex items-center gap-2 text-white/60">
                <div className="typing-indicator inline-flex">
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
                <span>讨论进行中...</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
