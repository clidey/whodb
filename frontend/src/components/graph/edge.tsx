/**
 * Copyright 2025 Clidey, Inc.
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

import { FC, useCallback } from 'react';
import { ConnectionLineComponentProps, EdgeProps, Node, Position, getBezierPath, getSimpleBezierPath, useStore } from 'reactflow';
import { getEdgeParams } from './utils';

export const GraphEdgeConnectionLine: FC<ConnectionLineComponentProps> = ({ toX, toY, fromPosition, toPosition, fromNode }) => {
  if (!fromNode) {
    return null;
  }

  const targetNode: Node = {
    id: 'connection-target',
    width: 1,
    height: 1,
    positionAbsolute: { x: toX, y: toY },
    data: {},
    position: { x: toX, y: toY, },
  };

  const { sx, sy } = getEdgeParams(fromNode, targetNode);
  const [edgePath] = getBezierPath({
    sourceX: sx,
    sourceY: sy,
    sourcePosition: fromPosition,
    targetPosition: toPosition,
    targetX: toX,
    targetY: toY
  });

  return (
      <g>
        <path
            fill="none"
            strokeWidth={1.5}
            className="stroke-neutral-600 dark:stroke-neutral-300 animated"
            d={edgePath}
        />
        <circle
            cx={toX}
            cy={toY}
            r={3}
            className="fill-white dark:fill-white/10 stroke-neutral-600 dark:stroke-neutral-300"
            strokeWidth={1.5}
        />
      </g>
  );
}

export const FloatingGraphEdge: FC<EdgeProps> = ({ id, source, target, sourceHandleId, targetHandleId, sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition, markerEnd, style }) => {
  const sourceNode = useStore(useCallback((store) => store.nodeInternals.get(source), [source]));
  const targetNode = useStore(useCallback((store) => store.nodeInternals.get(target), [target]));

  if (!sourceNode || !targetNode) {
    return null;
  }

  // If handles are specified and we have coordinates from React Flow, use them
  // Otherwise fallback to floating edge calculation
  let edgePath: string;

  if (sourceHandleId && targetHandleId && sourceX !== undefined && sourceY !== undefined && targetX !== undefined && targetY !== undefined) {
    // React Flow has calculated handle positions for us
    [edgePath] = getBezierPath({
      sourceX,
      sourceY,
      sourcePosition: sourcePosition || Position.Right,
      targetPosition: targetPosition || Position.Left,
      targetX,
      targetY,
    });
  } else {
    // Fallback to floating edge calculation
    const { sx, sy, tx, ty, sourcePos, targetPos } = getEdgeParams(sourceNode, targetNode);
    [edgePath] = getBezierPath({
      sourceX: sx,
      sourceY: sy,
      sourcePosition: sourcePos,
      targetPosition: targetPos,
      targetX: tx,
      targetY: ty,
    });
  }

  return (
    <path
      id={id}
      className="react-flow__edge-path"
      d={edgePath}
      markerEnd={markerEnd}
      style={style}
    />
  );
}