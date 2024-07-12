import classNames from "classnames";
import { motion } from "framer-motion";
import { FC, MouseEvent, ReactElement, ReactNode, cloneElement } from "react";
import { twMerge } from "tailwind-merge";

export type IButtonProps = {
  className?: string;
  label: string;
  icon: ReactElement;
  iconClassName?: string;
  labelClassName?: string;
  onClick?: (e: MouseEvent<HTMLButtonElement>, ...args: any) => void;
  disabled?: boolean;
  type?: "lg" | "sm";
}

export const Button: FC<IButtonProps> = (props) => {
  return <motion.button className={twMerge(classNames("rounded-lg border flex justify-center items-center text-xs px-2 py-1 cursor-pointer gap-1 bg-white hover:bg-gray-100 dark:bg-white/10 dark:hover:bg-white/15 dark:border-white/20 dark:backdrop-blur-md", props.className, {
    "cursor-not-allowed": props.disabled,
    "h-[35px] rounded-[4px] gap-2 hover:gap-3": props.type === "lg",
  }))} onClick={props.onClick} disabled={props.disabled} whileTap={{ scale: 0.8 }}>
    <div className={classNames("text-xs text-gray-600 dark:text-neutral-100", props.labelClassName)}>
      {props.label}
    </div>
    {cloneElement(props.icon, {
      className: classNames("w-4 h-4 stroke-gray-600 dark:stroke-white", props.iconClassName),
    })}
  </motion.button>
}

export const AnimatedButton: FC<IButtonProps> = (props) => {
  return <Button {...props} className={twMerge(classNames("transition-all hover:gap-2", props.className))} />
}


export type IActionButtonProps = {
  icon: ReactElement;
  className?: string;
  containerClassName?: string;
  onClick?: () => void;
  disabled?: boolean;
  children?: ReactNode;
}

export const ActionButton: FC<IActionButtonProps> = ({ onClick, icon, className, containerClassName, disabled, children }) => {
  return (
  <div className="group relative">
    <motion.button className={twMerge(classNames("rounded-full bg-white h-12 w-12 transition-all border border-gray-300 shadow-sm flex items-center justify-center", containerClassName, {
      "cursor-not-allowed": disabled,
      "hover:shadow-lg hover:cursor-pointer hover:scale-110": !disabled,
    }))} onClick={disabled ? undefined : onClick} whileTap={{ scale: 0.6, transition: { duration: 0.05 }, }}>
      {cloneElement(icon, {
          className: twMerge(classNames("w-8 h-8 stroke-gray-500 cursor-pointer", className))
      })}
    </motion.button>
    {children}
  </div>);
}