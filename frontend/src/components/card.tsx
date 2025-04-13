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

import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import {
  FC,
  ReactElement,
  ReactNode,
  cloneElement,
  memo,
  useEffect,
  useMemo,
  useState
} from "react";
import { twMerge } from "tailwind-merge";
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

export const Icon: FC<ICardIcon> = memo((propsIcon) => (<div
  className={twMerge(classNames(
    "h-[40px] w-[40px] rounded-lg flex justify-center items-center shadow-sm border dark:border-white/5",
    propsIcon.bgClassName
  ))}
>
  {cloneElement(propsIcon.component, {
    className: twMerge(classNames("w-6 h-6 stroke-white", propsIcon.className)),
  })}
</div>));

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
      setTimeout(() => {
        setHighlightStatus(false);
      }, 3000);
    }
  }, [highlight]);

  const icon = useMemo(() => {
      if (propsIcon == null) {
        return null;
      }
      if ("component" in propsIcon) {
        return <Icon {...propsIcon} />
      }
      return propsIcon;
  }, [propsIcon]);

  return (
    <motion.div
      className={twMerge(
        classNames(
          "bg-[#fbfaf8] h-[200px] w-[200px] rounded-3xl shadow-xs border border-neutral-600/5 p-4 flex flex-col justify-between relative transition-all duration-300 overflow-y-auto dark:bg-[#252525] dark:border-white/5",
          {
            "shadow-2xl z-10": highlightStatus,
          },
          className
        )
      )}
    >
      {loading ? (
        <Loading loadingText={loadingText} />
      ) : (
        <>
          <div className="flex justify-between items-start dark:text-neutral-300">
            {icon}
            {tag}
          </div>
          {children}
        </>
      )}
    </motion.div>
  );
};


type IExpandableCardProps = {
  isExpanded?: boolean;
  children: [ReactElement, ReactElement];
  setToggleCallback?: (callback: (status: boolean) => void) => void;
  collapsedTag?: ReactElement;
} & ICardProps;

export const ExpandableCard: FC<IExpandableCardProps> = (props) => {
  const [expand, setExpand] = useState(props.isExpanded);

  useEffect(() => {
    props.setToggleCallback?.(setExpand);
  }, [props]);

  useEffect(() => {
    setExpand(props.isExpanded);
  }, [props.isExpanded]);

  return (
    <AnimatePresence mode="sync">
      <Card
        {...props}
        className={twMerge(classNames({
          "w-full max-w-[400px] h-fit": expand,
        }, props.className))}
        tag={expand ? props.tag : props.collapsedTag}
      >
        <AnimatePresence mode="sync">
          <motion.div
            key={props.loading ? "loading" : expand ? "expand" : "collapse"}
            className="flex flex-col grow"
            initial={{ opacity: 0 }}
            animate={{ opacity: 100, transition: { duration: 0.5 } }}
            exit={{ opacity: 0, transition: { duration: 0.02 } }}
          >
            {props.loading ? (
              <Loading />
            ) : expand ? (
              props.children[1]
            ) : (
              props.children[0]
            )}
          </motion.div>
        </AnimatePresence>
      </Card>
    </AnimatePresence>
  );
};