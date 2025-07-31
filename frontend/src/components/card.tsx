/*
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

import {
  Card as UxCard,
  CardHeader,
  CardContent,
  Badge,
  cn,
  Sheet,
  SheetTrigger,
  SheetContent,
} from "@clidey/ux";
import {
  FC,
  ReactElement,
  ReactNode,
  cloneElement,
  memo,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Loading } from "./loading";


type ICardIcon = {
  component: ReactElement;
  bgClassName?: string;
  className?: string;
};

type ICardProps = {
  className?: string;
  icon?: ICardIcon | ReactElement;
  tag?: ReactElement;
  children: ReactElement[] | ReactElement | ReactNode;
  loading?: boolean;
  highlight?: boolean;
  loadingText?: string;
};

export const Icon: FC<ICardIcon> = memo((propsIcon) => (
  <div
    className={[
      "h-[40px] w-[40px] rounded-lg flex justify-center items-center",
      propsIcon.bgClassName,
    ]
      .filter(Boolean)
      .join(" ")}
  >
    {cloneElement(propsIcon.component, {
      className: ["w-6 h-6 stroke-white", propsIcon.className]
        .filter(Boolean)
        .join(" "),
    })}
  </div>
));

export const Card: FC<ICardProps> = ({
  children,
  className,
  icon: propsIcon,
  tag,
  highlight,
  loading,
  loadingText,
}) => {
  const [highlightStatus, setHighlightStatus] = useState(highlight);

  useEffect(() => {
    if (highlight) {
      setHighlightStatus(true);
      const timeout = setTimeout(() => {
        setHighlightStatus(false);
      }, 3000);
      return () => clearTimeout(timeout);
    }
  }, [highlight]);

  const icon = useMemo(() => {
    if (propsIcon == null) {
      return null;
    }
    if (typeof propsIcon === "object" && "component" in propsIcon) {
      return <Icon {...propsIcon} />;
    }
    return propsIcon;
  }, [propsIcon]);

  return (
    <UxCard
      className={cn(
        "py-4 gap-2",
        {
          "shadow-2xl z-10": highlightStatus,
        },
        className,
      )}
    >
      {loading ? (
        <Loading loadingText={loadingText} />
      ) : (
        <>
          {(icon || tag) && (
            <CardHeader className="flex flex-row justify-between items-start px-4">
              {icon}
              {tag && <Badge variant="secondary">{tag}</Badge>}
            </CardHeader>
          )}
          <CardContent className="px-4 grow">{children}</CardContent>
        </>
      )}
    </UxCard>
  );
};

type IExpandableCardProps = {
  isExpanded?: boolean;
  children: [ReactElement, ReactElement];
  setExpanded?: (status: boolean) => void;
  collapsedTag?: ReactElement;
} & ICardProps;

export const ExpandableCard: FC<IExpandableCardProps> = (props) => {
  const [expand, setExpand] = useState(props.isExpanded ?? false);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    setExpand(props.isExpanded ?? false);
  }, [props.isExpanded]);

  // Sheet expects controlled open/close
  const handleOpenChange = (open: boolean) => {
    setExpand(open);
    props.setExpanded?.(open);
  };

  // The collapsed card is always visible; clicking it opens the sheet
  return (
    <>
      <Sheet open={expand} onOpenChange={handleOpenChange}>
        <SheetTrigger asChild>
          <div>
            <Card
              {...props}
              tag={props.collapsedTag}
              className={cn(
                "min-h-[200px] w-full max-w-[400px] cursor-pointer",
                props.className,
              )}
              loading={props.loading}>
              {props.loading ? null : props.children[0]}
            </Card>
          </div>
        </SheetTrigger>
        <SheetContent side="right" className="p-0">
          <div className="flex flex-col w-full justify-center p-8 h-full">
            {props.loading ? null : props.children[1]}
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
};