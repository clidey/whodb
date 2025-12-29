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

import { EdgeMarker, MarkerType, Node, Position, Viewport, XYPosition } from "reactflow";

function getNodeIntersection(node: Node, otherNode: Node, padding = 1) {
  // Add fallbacks for missing dimensions and positions
  const nodePos = node.positionAbsolute || { x: 0, y: 0 };
  const otherPos = otherNode.positionAbsolute || { x: 0, y: 0 };
  const nodeWidth = node.width || 100;
  const nodeHeight = node.height || 50;
  const otherWidth = otherNode.width || 100;
  const otherHeight = otherNode.height || 50;

  const nodeCenterX = nodePos.x + nodeWidth / 2;
  const nodeCenterY = nodePos.y + nodeHeight / 2;
  const otherCenterX = otherPos.x + otherWidth / 2;
  const otherCenterY = otherPos.y + otherHeight / 2;

  const dx = otherCenterX - nodeCenterX;
  const dy = otherCenterY - nodeCenterY;

  // Avoid division by zero
  const scale = Math.max(
    Math.abs(dx) / ((nodeWidth + padding * 2) / 2),
    Math.abs(dy) / ((nodeHeight + padding * 2) / 2),
    1 // minimum scale to avoid division by zero
  );

  return {
    x: nodeCenterX + (dx / scale) - Math.sign(dx) * padding,
    y: nodeCenterY + (dy / scale) - Math.sign(dy) * padding,
  };
}

function getEdgePosition(node: Node, intersectionPoint: { x: number; y: number }) {
    // Add fallbacks for missing dimensions and positions
    const nodePos = node.positionAbsolute || { x: 0, y: 0 };
    const nodeWidth = node.width || 100;
    const nodeHeight = node.height || 50;

    const px = intersectionPoint.x;
    const py = intersectionPoint.y;

    const isLeft = Math.abs(px - nodePos.x) <= 1;
    const isRight = Math.abs(px - (nodePos.x + nodeWidth)) <= 1;
    const isTop = Math.abs(py - nodePos.y) <= 1;
    const isBottom = Math.abs(py - (nodePos.y + nodeHeight)) <= 1;

    if (isLeft) return Position.Left;
    if (isRight) return Position.Right;
    if (isTop) return Position.Top;
    if (isBottom) return Position.Bottom;

  return Position.Top;
}
  
export function getEdgeParams(source: Node, target: Node, padding = 10) {
  const sourceIntersectionPoint = getNodeIntersection(source, target);
  const targetIntersectionPoint = getNodeIntersection(target, source);

  const sourcePos = getEdgePosition(source, sourceIntersectionPoint);
  const targetPos = getEdgePosition(target, targetIntersectionPoint);

  const dx = targetIntersectionPoint.x - sourceIntersectionPoint.x;
  const dy = targetIntersectionPoint.y - sourceIntersectionPoint.y;
  const length = Math.sqrt(dx * dx + dy * dy);

  // Avoid division by zero when nodes are at the same position
  const paddingX = length > 0 ? (dx / length) * padding : 0;
  const paddingY = length > 0 ? (dy / length) * padding : 0;

  return {
    sx: sourceIntersectionPoint.x + paddingX,
    sy: sourceIntersectionPoint.y + paddingY,
    tx: targetIntersectionPoint.x - paddingX,
    ty: targetIntersectionPoint.y - paddingY,
    sourcePos,
    targetPos,
  };
}

export function createNode(node: Pick<Node, "id" | "type" | "data"> & Partial<Node>, position?: XYPosition,  viewPort?: Viewport): Node {
  return {
      ...node,
      position: {
          x: position?.x != null ? position.x - 100 : (viewPort?.x ?? 100) / 2,
          y: position?.y != null ? position.y : (viewPort?.y ?? 100) / 2,
      },
      className: "group-[.laying-out]:transition-all group-[.laying-out]:duration-300 -z-5",
  }
}

export const MARKER_END: EdgeMarker = {
  type: MarkerType.ArrowClosed,
  width: 30,
  height: 30,
  orient: 'auto-start-reverse',
};

/**
 * Creates an edge for the graph visualization.
 *
 * @param source - The source node ID
 * @param target - The target node ID
 * @param type - The edge type. Defaults to "floatingGraphEdge" which uses our custom FloatingGraphEdge
 *               component that includes data-edge-source and data-edge-target attributes for testing.
 * @returns Edge configuration object
 */
export function createEdge(source: string, target: string, type = "floatingGraphEdge") {
  return {
      id: `${source}->${target}`,
      type,
      source,
      target,
      markerEnd: MARKER_END,
      className: "transition-all duration-300 group-[.laying-out]:opacity-0",
  };
}
