import { PhilosopherType, PhilosopherInfo, PHILOSOPHERS } from '../types';

interface Props {
  selected: PhilosopherType;
  onSelect: (type: PhilosopherType) => void;
}

const colorMap: Record<string, string> = {
  tomori: 'from-violet-600 to-purple-700 border-violet-500',
  anon: 'from-amber-500 to-orange-600 border-amber-400',
  rana: 'from-emerald-500 to-green-600 border-emerald-400',
  soyo: 'from-pink-500 to-rose-600 border-pink-400',
  taki: 'from-blue-500 to-indigo-600 border-blue-400',
};

const bgColorMap: Record<string, string> = {
  tomori: 'bg-violet-500/20',
  anon: 'bg-amber-500/20',
  rana: 'bg-emerald-500/20',
  soyo: 'bg-pink-500/20',
  taki: 'bg-blue-500/20',
};

export function CharacterSelect({ selected, onSelect }: Props) {
  return (
    <div className="grid grid-cols-5 gap-3">
      {PHILOSOPHERS.map((char) => (
        <button
          key={char.type}
          onClick={() => onSelect(char.type)}
          className={`character-card relative p-4 rounded-xl border-2 transition-all ${
            selected === char.type
              ? `${colorMap[char.color]} bg-gradient-to-br border-2 shadow-lg`
              : `border-white/10 hover:border-white/30 ${bgColorMap[char.color]}`
          }`}
        >
          <div className="text-3xl mb-2">{char.avatar}</div>
          <div className="font-bold text-sm">{char.name}</div>
          <div className="text-xs text-white/60">{char.role}</div>
          {selected === char.type && (
            <div className="absolute -top-1 -right-1 w-4 h-4 bg-white rounded-full flex items-center justify-center">
              <span className="text-xs">✓</span>
            </div>
          )}
        </button>
      ))}
    </div>
  );
}

export function CharacterCard({ char, isSelected, onClick }: { 
  char: PhilosopherInfo; 
  isSelected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`character-card w-full p-4 rounded-xl border-2 text-left transition-all ${
        isSelected
          ? `${colorMap[char.color]} bg-gradient-to-br shadow-lg`
          : `border-white/10 hover:border-white/30 ${bgColorMap[char.color]}`
      }`}
    >
      <div className="flex items-center gap-3">
        <span className="text-4xl">{char.avatar}</span>
        <div>
          <div className="font-bold">{char.name}</div>
          <div className="text-sm text-white/60">{char.nameJp}</div>
          <div className="text-xs text-white/50 mt-1">{char.role}</div>
        </div>
      </div>
      <p className="text-sm text-white/70 mt-3">{char.description}</p>
    </button>
  );
}
