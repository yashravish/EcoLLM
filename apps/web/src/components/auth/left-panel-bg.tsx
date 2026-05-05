const PARTICLES: Array<{
  left: string; top: string;
  w: number; h: number;
  delay: string; dur: string;
}> = [
  { left: '10%', top: '58%', w: 2,   h: 2,   delay: '0s',    dur: '9s'   },
  { left: '27%', top: '67%', w: 1.5, h: 1.5, delay: '2.5s',  dur: '11s'  },
  { left: '44%', top: '74%', w: 2,   h: 2,   delay: '1.2s',  dur: '8s'   },
  { left: '61%', top: '63%', w: 1.5, h: 1.5, delay: '4.2s',  dur: '10s'  },
  { left: '77%', top: '78%', w: 2.5, h: 2.5, delay: '3s',    dur: '12s'  },
  { left: '36%', top: '85%', w: 1,   h: 1,   delay: '6.5s',  dur: '9.5s' },
  { left: '53%', top: '55%', w: 1.5, h: 1.5, delay: '0.8s',  dur: '7.5s' },
  { left: '19%', top: '80%', w: 1,   h: 1,   delay: '5s',    dur: '10.5s'},
];

export function LeftPanelBg({ orbs = true }: { orbs?: boolean }) {
  return (
    <div className="absolute inset-0 pointer-events-none overflow-hidden" aria-hidden="true">
      {orbs && <div className="lp-orb lp-orb-1" />}
      {orbs && <div className="lp-orb lp-orb-2" />}
      {PARTICLES.map((p, i) => (
        <div
          key={i}
          className="lp-particle"
          style={{
            left: p.left,
            top: p.top,
            width: p.w,
            height: p.h,
            animationDelay: p.delay,
            animationDuration: p.dur,
          }}
        />
      ))}
    </div>
  );
}
