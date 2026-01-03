import { useRef, useEffect } from 'react';
import { PhilosopherType, PHILOSOPHERS } from '../types';
import { useChat } from '../hooks/useChat';
import { CharacterSelect } from './CharacterSelect';
import { ChatMessage, TypingIndicator } from './ChatMessage';
import { ChatInput } from './ChatInput';

interface Props {
  selectedChar: PhilosopherType;
  onSelectChar: (type: PhilosopherType) => void;
}

export function ChatView({ selectedChar, onSelectChar }: Props) {
  const { messages, isLoading, sendMessage, clearMessages } = useChat();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const currentChar = PHILOSOPHERS.find(p => p.type === selectedChar)!;

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async (content: string) => {
    try {
      await sendMessage(content, selectedChar);
    } catch (error) {
      console.error('Send failed:', error);
    }
  };

  const handleCharChange = (type: PhilosopherType) => {
    if (type !== selectedChar) {
      onSelectChar(type);
      clearMessages();
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* 头部 - 角色选择 */}
      <div className="p-4 border-b border-white/10 bg-black/20 backdrop-blur-sm">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="text-3xl">{currentChar.avatar}</span>
            <div>
              <h2 className="font-bold text-lg">{currentChar.name}</h2>
              <p className="text-sm text-white/60">{currentChar.nameJp}</p>
            </div>
          </div>
          <button
            onClick={clearMessages}
            className="px-3 py-1.5 text-sm bg-white/10 hover:bg-white/20 rounded-lg transition-colors"
          >
            清空对话
          </button>
        </div>
        <CharacterSelect selected={selectedChar} onSelect={handleCharChange} />
      </div>

      {/* 消息列表 */}
      <div className="flex-1 overflow-y-auto p-4">
        {messages.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-center">
            <span className="text-6xl mb-4">{currentChar.avatar}</span>
            <h3 className="text-xl font-bold mb-2">和 {currentChar.name} 聊天</h3>
            <p className="text-white/60 max-w-md">{currentChar.description}</p>
            <p className="text-white/40 text-sm mt-4">发送消息开始对话吧</p>
          </div>
        ) : (
          <>
            {messages.map((msg) => (
              <ChatMessage key={msg.id} message={msg} />
            ))}
            {isLoading && <TypingIndicator />}
            <div ref={messagesEndRef} />
          </>
        )}
      </div>

      {/* 输入框 */}
      <ChatInput
        onSend={handleSend}
        disabled={isLoading}
        placeholder={`对 ${currentChar.name} 说点什么...`}
      />
    </div>
  );
}
