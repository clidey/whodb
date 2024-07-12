import { FC, useCallback } from 'react';
import { ConnectionLineComponentProps, EdgeProps, Node, getBezierPath, useStore } from 'reactflow';
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
            className="stroke-neutral-800 dark:stroke-neutral-300 animated"
            strokeWidth={1.5}
            d={edgePath}
        />
        <circle
            className="fill-white stroke-neutral-800 dark:stroke-neutral-300"
            cx={toX}
            cy={toY}
            r={3}
            strokeWidth={1.5}
        />
      </g>
  );
}

export const FloatingGraphEdge: FC<EdgeProps> = ({ id, source, target, markerEnd, style }) => {
  const sourceNode = useStore(useCallback((store) => store.nodeInternals.get(source), [source]));
  const targetNode = useStore(useCallback((store) => store.nodeInternals.get(target), [target]));

  if (!sourceNode || !targetNode) {
    return null;
  }

  const { sx, sy, tx, ty, sourcePos, targetPos } = getEdgeParams(sourceNode, targetNode);

  const [edgePath] = getBezierPath({
    sourceX: sx,
    sourceY: sy,
    sourcePosition: sourcePos,
    targetPosition: targetPos,
    targetX: tx,
    targetY: ty,
  });

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