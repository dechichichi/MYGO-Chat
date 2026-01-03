import { Message } from '../types';

interface Props {
  message: Message;
}

export function ChatMessage({ message }: Props) {
  const isUser = message.role === 'user';

  return (
    <div className={`message-bubble flex ${isUser ? 'justify-end' : 'justify-start'} mb-4`}>
      <div
        className={`max-w-[80%] rounded-2xl px-4 py-3 ${
          isUser
            ? 'bg-gradient-to-r from-pink-500 to-rose-500 text-white'
            : 'bg-white/10 backdrop-blur-sm border border-white/10'
        }`}
      >
        {!isUser && message.philosopher && (
          <div className="text-xs text-white/50 mb-1">{message.philosopher}</div>
        )}
        <div className="whitespace-pre-wrap leading-relaxed">{message.content}</div>
        <div className={`text-xs mt-2 ${isUser ? 'text-white/70' : 'text-white/40'}`}>
          {message.timestamp.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
        </div>
      </div>
    </div>
  );
}

export function TypingIndicator() {
  return (
    <div className="flex justify-start mb-4">
      <div className="bg-white/10 backdrop-blur-sm border border-white/10 rounded-2xl">
        <div className="typing-indicator">
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>
    </div>
  );
}
