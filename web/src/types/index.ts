export type PhilosopherType = 'tomori' | 'anon' | 'rana' | 'soyo' | 'taki';

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  philosopher?: string;
}

export interface ChatResponse {
  response: string;
  philosopher: string;
  emotion_level: string;
  critical_hit: boolean;
}

export interface DebateRecord {
  speaker_name: string;
  content: string;
  phase: string;
}

export interface DebateResponse {
  id?: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  topic?: string;
  current_phase?: string;
  records?: DebateRecord[];
  error?: string;
}

export interface PhilosopherInfo {
  type: PhilosopherType;
  name: string;
  nameJp: string;
  role: string;
  color: string;
  description: string;
  avatar: string;
}

export const PHILOSOPHERS: PhilosopherInfo[] = [
  {
    type: 'tomori',
    name: '高松灯',
    nameJp: 'Takamatsu Tomori',
    role: '主唱',
    color: 'tomori',
    description: '感性细腻的"羽丘怪女生"，用诗意的语言表达内心',
    avatar: '🎤',
  },
  {
    type: 'anon',
    name: '千早爱音',
    nameJp: 'Chihaya Anon',
    role: '吉他',
    color: 'anon',
    description: '元气满满的优等生，想要闪闪发光',
    avatar: '🎸',
  },
  {
    type: 'rana',
    name: '要乐奈',
    nameJp: 'Kaname Rana',
    role: '鼓手',
    color: 'rana',
    description: '神出鬼没的古怪少女，觉得一切都很有趣',
    avatar: '🥁',
  },
  {
    type: 'soyo',
    name: '长崎素世',
    nameJp: 'Nagasaki Soyo',
    role: '贝斯',
    color: 'soyo',
    description: '温柔的大姐姐，内心渴望真正的连接',
    avatar: '🎻',
  },
  {
    type: 'taki',
    name: '椎名立希',
    nameJp: 'Shiina Taki',
    role: '吉他',
    color: 'taki',
    description: '傲娇的独狼，嘴硬心软的乐队实际领导者',
    avatar: '🎵',
  },
];
