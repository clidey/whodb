/**
 * Motion model for the 404 page: the WhoDB mark runs at the "404", hits it,
 * squashes, rebounds, wobbles dazed, rests, fades and loops. Every value is a
 * pure function of the loop clock so the page writes it straight to the DOM.
 *
 * Phases (seconds into the 5.4 s loop):
 *   run 0–1.30 · impact 1.30–1.46 · rebound 1.46–2.16 · dazed 2.16–3.96 ·
 *   rest 3.96–4.60 · fade 4.60–5.10 · hidden until 5.40
 *
 * The auth-host static page (ee/infra_gcp maintenance nginx) carries a plain
 * JavaScript copy of this function; keep the two in step.
 */

/** Length of one animation loop in seconds. */
export const LOOP_SECONDS = 5.4;

/** Loop time of the resting pose, used for reduced motion. */
export const REST_TIME = 4.0;

/** Rendered width of the mark in CSS pixels (viewBox 0 0 200 160 at 140 × 112). */
export const MARK_WIDTH = 140;

/** How far the mark's right edge overlaps the "404" box at contact. */
export const CONTACT_OVERLAP = 4;

const RUN_END = 1.3;
const IMPACT_END = 0.16;
const REBOUND_END = 0.86;
const DAZED_SECONDS = 1.8;
const FADE_START = 4.6;
const FADE_END = 5.1;

/** One frame of the animation. Lengths are CSS pixels, angles degrees. */
export type Pose = {
  /** Outer wrapper translate; 0,0 is the contact position. */
  x: number;
  y: number;
  /** Outer wrapper rotation around the stem foot (origin 50% 95%). */
  rotate: number;
  /** Inner wrapper squash, anchored on the face that hits the wall (origin 100% 50%). */
  scaleX: number;
  scaleY: number;
  opacity: number;
  /** Vertical offset of the yellow head from its seated position. */
  headOffset: number;
  /** False while running (flowing dashes), true from impact onward. */
  solid: boolean;
  /** Dash offsets for the flowing signals while running. */
  armDashOffset: number;
  stemDashOffset: number;
  /** Opacity of the four rings: left end, right end, left mid, right mid. */
  rings: [number, number, number, number];
  /** Horizontal shake of the "404" box. */
  wallShake: number;
  impactOpacity: number;
  impactScale: number;
};

const easeInOutCubic = (u: number): number => (u < 0.5 ? 4 * u * u * u : 1 - Math.pow(-2 * u + 2, 3) / 2);

/**
 * Returns the pose at loop time t for a mark whose contact position sits
 * `contact` pixels from the left edge of the stage.
 */
export function pose(t: number, contact: number): Pose {
  // Snap to microseconds so a keyframe time like 1.3 lands on its phase boundary.
  const p = Math.round((((t % LOOP_SECONDS) + LOOP_SECONDS) % LOOP_SECONDS) * 1e6) / 1e6;
  const frame: Pose = {
    x: 0, y: 0, rotate: 0, scaleX: 1, scaleY: 1, opacity: 1, headOffset: 0,
    solid: true, armDashOffset: 0.5, stemDashOffset: 0,
    rings: [1, 1, 1, 1], wallShake: 0, impactOpacity: 0, impactScale: 1,
  };

  if (p < RUN_END) {
    const u = p / RUN_END;
    const flow = Math.min(p / 0.9, 1);
    frame.x = -(contact + 160) * (1 - u * u);
    frame.y = -Math.abs(Math.sin(u * 5 * Math.PI)) * 7;
    frame.rotate = 7;
    frame.opacity = Math.min(p / 0.25, 1);
    frame.solid = false;
    frame.armDashOffset = 0.3 + 0.2 * flow;
    frame.stemDashOffset = frame.armDashOffset - 0.5;
    const endRing = p <= 0.9 ? 0.4 + 0.6 * (0.5 + 0.5 * flow) : 0.4;
    frame.rings = [endRing, endRing, 0.4, 0.4];
    return frame;
  }

  const d = p - RUN_END;
  frame.wallShake = 7 * Math.exp(-7 * d) * Math.sin(55 * d);
  frame.impactOpacity = Math.max(0, 1 - d / 0.6);
  frame.impactScale = 0.7 + 0.3 * Math.min(d / IMPACT_END, 1);

  if (d < IMPACT_END) {
    frame.scaleX = 1 - 0.2 * Math.sin((Math.PI * d) / IMPACT_END);
    frame.scaleY = 2 - frame.scaleX;
    frame.rotate = 7 * (1 - d / IMPACT_END);
    return frame;
  }

  if (d < REBOUND_END) {
    const u = (d - IMPACT_END) / (REBOUND_END - IMPACT_END);
    const easeOut = 1 - Math.pow(1 - u, 3);
    frame.x = -84 * easeOut;
    frame.rotate = -20 * easeOut;
    frame.y = -26 * 4 * u * (1 - u);
    frame.headOffset = -46 * Math.sin(Math.PI * Math.min(1.3 * u, 1));
    return frame;
  }

  const e = d - REBOUND_END;
  frame.x = -84;
  frame.rotate = -20 + 20 * easeInOutCubic(Math.min(e / DAZED_SECONDS, 1)) + 2.5 * Math.exp(-1.5 * e) * Math.sin(14 * e);
  frame.headOffset = -10 * Math.exp(-6 * e) * Math.abs(Math.sin(14 * e));
  if (e < DAZED_SECONDS) {
    frame.rings = [0, 1, 2, 3].map(i => (Math.sin(9 * e - 1.7 * i) > 0.2 ? 1 : 0.35)) as Pose['rings'];
  }
  if (p >= FADE_START) {
    frame.opacity = p < FADE_END ? 1 - (p - FADE_START) / (FADE_END - FADE_START) : 0;
  }
  return frame;
}
