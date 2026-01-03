import { useState } from 'react';
import { PhilosopherType } from './types';
import { ChatView } from './components/ChatView';
import { DebateView } from './components/DebateView';

type ViewMode = 'chat' | 'debate';

function App() {
  const [viewMode, setViewMode] = useState<ViewMode>('chat');
  const [selectedChar, setSelectedChar] = useState<PhilosopherType>('tomori');

  return (
    <div className="min-h-screen flex flex-col">
      {/* 顶部导航 */}
      <header className="bg-black/30 backdrop-blur-sm border-b border-white/10">
        <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-2xl">🎸</span>
            <div>
              <h1 className="font-bold text-lg">MyGO!!!!! Chat</h1>
              <p className="text-xs text-white/50">迷子でもいい、迷子でも進め</p>
            </div>
          </div>

          {/* 模式切换 */}
          <div className="flex bg-white/10 rounded-lg p-1">
            <button
              onClick={() => setViewMode('chat')}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                viewMode === 'chat'
                  ? 'bg-pink-500 text-white'
                  : 'text-white/70 hover:text-white'
              }`}
            >
              💬 一对一聊天
            </button>
            <button
              onClick={() => setViewMode('debate')}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                viewMode === 'debate'
                  ? 'bg-pink-500 text-white'
                  : 'text-white/70 hover:text-white'
              }`}
            >
              🎤 乐队讨论
            </button>
          </div>
        </div>
      </header>

      {/* 主内容区 */}
      <main className="flex-1 max-w-6xl w-full mx-auto">
        <div className="h-[calc(100vh-72px)] bg-black/20 backdrop-blur-sm border-x border-white/10">
          {viewMode === 'chat' ? (
            <ChatView selectedChar={selectedChar} onSelectChar={setSelectedChar} />
          ) : (
            <DebateView />
          )}
        </div>
      </main>
    </div>
  );
}

export default App;
