type BrandMarkProps = {
  className?: string;
};

const center = 512;
const arms = Array.from({ length: 8 }, (_, index) => {
  const angle = (index * Math.PI) / 4;
  const directionX = Math.cos(angle);
  const directionY = Math.sin(angle);
  const normalX = -directionY;
  const normalY = directionX;
  const radius = index % 2 === 0 ? 344 : 314;
  const base = 40;
  const tip = 15;

  const points: Array<[number, number]> = [
    [center + normalX * base, center + normalY * base],
    [center + directionX * radius + normalX * tip, center + directionY * radius + normalY * tip],
    [center + directionX * radius - normalX * tip, center + directionY * radius - normalY * tip],
    [center - normalX * base, center - normalY * base],
  ];

  return {
    points: points.map(([x, y]) => `${x.toFixed(2)},${y.toFixed(2)}`).join(" "),
    tipX: center + directionX * radius,
    tipY: center + directionY * radius,
  };
});

export function BrandMark({ className }: BrandMarkProps) {
  return (
    <svg
      className={className}
      viewBox="0 0 1024 1024"
      fill="currentColor"
      aria-hidden="true"
      focusable="false"
    >
      {arms.map((arm) => (
        <g key={arm.points}>
          <polygon points={arm.points} />
          <circle cx={arm.tipX} cy={arm.tipY} r={15} />
        </g>
      ))}
    </svg>
  );
}
