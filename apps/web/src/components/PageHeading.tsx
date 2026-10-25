import { useEffect, useRef, type ReactNode } from 'react';
/** As páginas são remontadas pela rota; foco consistente inclusive durante loading/erro. */
export function PageHeading({ children }: { children: ReactNode }) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus(); }, []);
  return <h2 ref={heading} tabIndex={-1}>{children}</h2>;
}
