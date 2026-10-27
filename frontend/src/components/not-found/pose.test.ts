/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {LOOP_SECONDS, REST_TIME, pose} from './pose';

// Keyframes from the WhoDB 404 motion spec, rendered with contact = 72.
// Plain script, run by scripts/run-not-found-tests.mjs (no test framework).
const CONTACT = 72;

const assert = {
  ok: (condition: boolean, message: string) => {
    if (!condition) throw new Error(message);
  },
  equal: (actual: unknown, expected: unknown) => {
    if (!Object.is(actual, expected)) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  },
  deepEqual: (actual: unknown, expected: unknown) => {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  },
};
const near = (actual: number, expected: number, tolerance: number, what: string) => {
  assert.ok(Math.abs(actual - expected) <= tolerance, `${what}: ${actual} not within ${tolerance} of ${expected}`);
};

let failures = 0;
const it = (name: string, run: () => void) => {
  try {
    run();
  } catch (error) {
    failures++;
    console.error(`not ok - ${name}\n    ${(error as Error).message}`);
  }
};
const describe = (_name: string, run: () => void) => {
  run();
};

describe('404 pose', () => {
  it('runs in from off stage, leaning forward and fading up (F01)', () => {
    const f = pose(0.1, CONTACT);
    near(f.x, -230.6, 0.5, 'x');
    near(f.y, -6.5, 0.5, 'y');
    assert.equal(f.rotate, 7);
    near(f.opacity, 0.4, 0.01, 'opacity');
    assert.equal(f.solid, false);
    near(f.rings[0], 0.7, 0.05, 'end ring');
    assert.equal(f.rings[2], 0.4);
  });

  it('touches the wall solid and square at contact (F04)', () => {
    const f = pose(1.3, CONTACT);
    assert.equal(f.x, 0);
    assert.equal(f.scaleX, 1);
    assert.equal(f.solid, true);
    assert.deepEqual(f.rings, [1, 1, 1, 1]);
    assert.equal(f.impactOpacity, 1);
  });

  it('squashes to 0.8 × 1.2 at peak impact and shakes the 404 (F05)', () => {
    const f = pose(1.38, CONTACT);
    near(f.scaleX, 0.8, 0.01, 'scaleX');
    near(f.scaleY, 1.2, 0.01, 'scaleY');
    near(f.rotate, 3.5, 0.1, 'rotate');
    near(f.wallShake, -3.8, 0.1, 'shake');
    near(f.impactOpacity, 0.87, 0.01, 'impact opacity');
  });

  it('hops back with the head popping up at the rebound apex (F07)', () => {
    const f = pose(1.78, CONTACT);
    near(f.x, -70.6, 0.5, 'x');
    near(f.y, -25.8, 0.5, 'y');
    near(f.rotate, -16.8, 0.5, 'rotate');
    near(f.headOffset, -44, 0.5, 'head');
    near(f.impactOpacity, 0.2, 0.05, 'impact opacity');
  });

  it('lands 84 px out, tilted back, with two rings dimmed (F08)', () => {
    const f = pose(2.16, CONTACT);
    assert.equal(f.x, -84);
    near(f.rotate, -20, 0.1, 'rotate');
    near(f.headOffset, 0, 0.1, 'head');
    assert.deepEqual(f.rings, [0.35, 0.35, 1, 1]);
  });

  it('rests upright and steady (F10), the reduced-motion pose', () => {
    const f = pose(REST_TIME, CONTACT);
    assert.equal(f.x, -84);
    assert.ok(Math.abs(f.rotate) < 0.5, `rotate ${f.rotate}`);
    assert.deepEqual(f.rings, [1, 1, 1, 1]);
    assert.equal(f.opacity, 1);
    assert.ok(Math.abs(f.wallShake) < 0.01, `shake ${f.wallShake}`);
  });

  it('fades out without moving, then loops hidden back to the start', () => {
    const fade = pose(4.85, CONTACT);
    near(fade.opacity, 0.5, 0.01, 'opacity');
    assert.equal(fade.x, -84);
    assert.equal(pose(5.2, CONTACT).opacity, 0);
    const loop = pose(LOOP_SECONDS, CONTACT);
    assert.equal(loop.opacity, 0);
    near(loop.x, -(CONTACT + 160), 0.5, 'loop x');
  });
});

if (failures > 0) {
  throw new Error(`${failures} failing`);
}
