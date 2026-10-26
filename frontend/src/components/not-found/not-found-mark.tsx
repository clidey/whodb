import {useEffect, useRef, type FC} from 'react';
import {CONTACT_OVERLAP, LOOP_SECONDS, MARK_WIDTH, REST_TIME, pose, type Pose} from './pose';

type NotFoundMarkProps = {
  /** Accessible label for the whole scene; the "404" is also readable text. */
  label: string;
};

/**
 * The animated 404 scene: the WhoDB mark runs at the "404" and bounces off it.
 * All motion comes from `pose()` on one requestAnimationFrame clock and is
 * written straight to the DOM. Under `prefers-reduced-motion` the resting pose
 * is shown with nothing animating.
 */
export const NotFoundMark: FC<NotFoundMarkProps> = ({label}) => {
  const stageRef = useRef<HTMLDivElement>(null);
  const outerRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const wallRef = useRef<HTMLDivElement>(null);
  const headRef = useRef<SVGCircleElement>(null);
  const impactRef = useRef<SVGGElement>(null);
  const armsRef = useRef<SVGGElement>(null);
  const stemRef = useRef<SVGPathElement>(null);
  const ringRefs = useRef<Array<SVGCircleElement | null>>([]);

  useEffect(() => {
    const stage = stageRef.current;
    const outer = outerRef.current;
    const inner = innerRef.current;
    const wall = wallRef.current;
    if (!stage || !outer || !inner || !wall) return;

    const apply = (frame: Pose, contact: number) => {
      outer.style.left = `${contact}px`;
      outer.style.opacity = String(frame.opacity);
      outer.style.transform = `translate(${frame.x}px, ${frame.y}px) rotate(${frame.rotate}deg)`;
      inner.style.transform = `scale(${frame.scaleX}, ${frame.scaleY})`;
      wall.style.transform = `translateX(${frame.wallShake}px)`;
      headRef.current?.setAttribute('cy', String(20 + frame.headOffset));
      if (armsRef.current) {
        armsRef.current.setAttribute('stroke-dasharray', frame.solid ? '1 0' : '0.5 1.5');
        armsRef.current.setAttribute('stroke-dashoffset', frame.solid ? '0' : String(frame.armDashOffset));
      }
      if (stemRef.current) {
        stemRef.current.setAttribute('stroke-dasharray', frame.solid ? '1 0' : '0.5 1.5');
        stemRef.current.setAttribute('stroke-dashoffset', frame.solid ? '0' : String(frame.stemDashOffset));
      }
      frame.rings.forEach((opacity, i) => {
        ringRefs.current[i]?.setAttribute('opacity', String(opacity));
      });
      if (impactRef.current) {
        impactRef.current.setAttribute('opacity', String(frame.impactOpacity));
        impactRef.current.setAttribute('transform', `translate(204 30) scale(${frame.impactScale})`);
      }
    };
    // The mark's right edge sits CONTACT_OVERLAP px inside the "404" box;
    // measured every frame because the 404 has a responsive font size.
    const contactAt = () => wall.offsetLeft - MARK_WIDTH + CONTACT_OVERLAP;

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      apply(pose(REST_TIME, contactAt()), contactAt());
      return;
    }
    let raf = 0;
    const start = performance.now();
    const tick = (now: number) => {
      const t = ((now - start) / 1000) % LOOP_SECONDS;
      const contact = contactAt();
      apply(pose(t, contact), contact);
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => {
      cancelAnimationFrame(raf);
    };
  }, []);

  return (
    <div ref={stageRef} className="whodb-404-stage" role="img" aria-label={label}>
      <div ref={outerRef} className="whodb-404-mark" aria-hidden="true">
        <div ref={innerRef} className="whodb-404-mark__squash">
          <svg viewBox="0 0 200 160" width={MARK_WIDTH} height={112} fill="none" overflow="visible">
            <g className="whodb-404-track" strokeWidth="4.5" strokeLinecap="round">
              <path d="M10 22 Q55 118 100 118" />
              <path d="M190 22 Q145 118 100 118" />
              <path d="M100 40 V152" className="whodb-404-track--stem" />
            </g>
            <g ref={armsRef} className="whodb-404-arm" strokeWidth="4.5" strokeLinecap="round" pathLength="1">
              <path d="M10 22 Q55 118 100 118" pathLength="1" />
              <path d="M190 22 Q145 118 100 118" pathLength="1" />
            </g>
            <path ref={stemRef} d="M100 40 V152" className="whodb-404-stem" strokeWidth="4.5" strokeLinecap="round" pathLength="1" />
            <circle ref={el => { ringRefs.current[0] = el; }} className="whodb-404-ring" cx="10" cy="22" r="8" strokeWidth="4" />
            <circle ref={el => { ringRefs.current[1] = el; }} className="whodb-404-ring" cx="190" cy="22" r="8" strokeWidth="4" />
            <circle ref={el => { ringRefs.current[2] = el; }} className="whodb-404-ring" cx="46" cy="83" r="5.5" strokeWidth="3.5" />
            <circle ref={el => { ringRefs.current[3] = el; }} className="whodb-404-ring" cx="154" cy="83" r="5.5" strokeWidth="3.5" />
            <circle ref={headRef} className="whodb-404-head" cx="100" cy="20" r="12" />
            <g ref={impactRef} className="whodb-404-impact" strokeWidth="4" strokeLinecap="round" opacity="0" transform="translate(204 30)">
              <path d="M-6 -22 L-14 -42" />
              <path d="M2 -26 L2 -48" />
              <path d="M10 -22 L20 -40" />
            </g>
          </svg>
        </div>
      </div>
      <div ref={wallRef} className="whodb-404-wall">404</div>
    </div>
  );
};
