import { useState, KeyboardEvent } from 'react';

interface Props {
  onSend: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export function ChatInput({ onSend, disabled, placeholder = '输入消息...' }: Props) {
  const [input, setInput] = useState('');

  const handleSend = () => {
    if (input.trim() && !disabled) {
      onSend(input.trim());
      setInput('');
    }
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex gap-3 p-4 bg-black/20 backdrop-blur-sm border-t border-white/10">
      <textarea
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={disabled}
        rows={1}
        className="flex-1 bg-white/10 border border-white/20 rounded-xl px-4 py-3 text-white placeholder-white/40 focus:outline-none focus:border-pink-500 resize-none"
        style={{ minHeight: '48px', maxHeight: '120px' }}
      />
      <button
        onClick={handleSend}
        disabled={disabled || !input.trim()}
        className="px-6 py-3 bg-gradient-to-r from-pink-500 to-rose-500 rounded-xl font-medium hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
      >
        发送
      </button>
    </div>
  );
}
