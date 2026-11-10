import type { ReactNode } from "react";

type WelcomeProps = {
  greeting: string;
  children: ReactNode;
};

export function Welcome({ greeting, children }: WelcomeProps) {
  return (
    <div className="welcome">
      <div className="welcome-inner">
        <h1>{greeting}</h1>
        {children}
      </div>
    </div>
  );
}
