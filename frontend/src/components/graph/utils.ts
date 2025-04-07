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
  const nodeCenterX = node.positionAbsolute!.x + node.width! / 2;
  const nodeCenterY = node.positionAbsolute!.y + node.height! / 2;
  const otherCenterX = otherNode.positionAbsolute!.x + otherNode.width! / 2;
  const otherCenterY = otherNode.positionAbsolute!.y + otherNode.height! / 2;

  const dx = otherCenterX - nodeCenterX;
  const dy = otherCenterY - nodeCenterY;

  const scale = Math.max(
    Math.abs(dx) / ((node.width! + padding * 2) / 2),
    Math.abs(dy) / ((node.height! + padding * 2) / 2)
  );

  return {
    x: nodeCenterX + (dx / scale) - Math.sign(dx) * padding,
    y: nodeCenterY + (dy / scale) - Math.sign(dy) * padding,
  };
}

function getEdgePosition(node: Node, intersectionPoint: { x: number; y: number }) {
    const nodeX = node.positionAbsolute!.x;
    const nodeY = node.positionAbsolute!.y;
    const nodeWidth = node.width!;
    const nodeHeight = node.height!;

    const px = intersectionPoint.x;
    const py = intersectionPoint.y;

    const isLeft = Math.abs(px - nodeX) <= 1;
    const isRight = Math.abs(px - (nodeX + nodeWidth)) <= 1;
    const isTop = Math.abs(py - nodeY) <= 1;
    const isBottom = Math.abs(py - (nodeY + nodeHeight)) <= 1;

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

  const paddingX = (dx / length) * padding;
  const paddingY = (dy / length) * padding;

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
