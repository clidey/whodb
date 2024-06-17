import { MarkerType, Node, Position, Viewport, XYPosition } from "reactflow";

function getNodeIntersection(intersectionNode: Node, targetNode: Node) {
    const {
      width: intersectionNodeWidth,
      height: intersectionNodeHeight,
      positionAbsolute: intersectionNodePosition,
    } = intersectionNode;
    const targetPosition = targetNode.positionAbsolute;
  
    const w = intersectionNodeWidth! / 2;
    const h = intersectionNodeHeight! / 2;
  
    const x2 = intersectionNodePosition!.x + w;
    const y2 = intersectionNodePosition!.y + h;
    const x1 = targetPosition!.x + targetNode.width! / 2;
    const y1 = targetPosition!.y + targetNode.height! / 2;
  
    const xx1 = (x1 - x2) / (2 * w) - (y1 - y2) / (2 * h);
    const yy1 = (x1 - x2) / (2 * w) + (y1 - y2) / (2 * h);
    const a = 1 / (Math.abs(xx1) + Math.abs(yy1));
    const xx3 = a * xx1;
    const yy3 = a * yy1;
    const x = w * (xx3 + yy3) + x2;
    const y = h * (-xx3 + yy3) + y2;
  
    return { x, y };
  }
  
  function getEdgePosition(node: Node, intersectionPoint: { x: number, y: number }) {
    const n = { ...node.positionAbsolute, ...node };
    const nx = Math.round(n.x!);
    const ny = Math.round(n.y!);
    const px = Math.round(intersectionPoint.x);
    const py = Math.round(intersectionPoint.y);
  
    if (px <= nx + 1) {
      return Position.Left;
    }
    if (px >= nx + n.width! - 1) {
      return Position.Right;
    }
    if (py <= ny + 1) {
      return Position.Top;
    }
    if (py >= n.y! + n.height! - 1) {
      return Position.Bottom;
    }
  
    return Position.Top;
  }
  
  export function getEdgeParams(source: Node, target: Node) {
    const sourceIntersectionPoint = getNodeIntersection(source, target);
    const targetIntersectionPoint = getNodeIntersection(target, source);
  
    const sourcePos = getEdgePosition(source, sourceIntersectionPoint);
    const targetPos = getEdgePosition(target, targetIntersectionPoint);
  
    return {
      sx: sourceIntersectionPoint.x,
      sy: sourceIntersectionPoint.y,
      tx: targetIntersectionPoint.x,
      ty: targetIntersectionPoint.y,
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

export const MARKER_END = {
  type: MarkerType.Arrow,
  width: 30,
  height: 30,
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
