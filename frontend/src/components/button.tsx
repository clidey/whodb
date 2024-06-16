import classNames from "classnames";
import { motion } from "framer-motion";
import { FC, ReactElement, cloneElement } from "react";
import { twMerge } from "tailwind-merge";

export type IButtonProps = {
  className?: string;
  label: string;
  icon: ReactElement;
  iconClassName?: string;
  labelClassName?: string;
  onClick?: () => void;
  disabled?: boolean;
  type?: "lg" | "sm";
}

export const Button: FC<IButtonProps> = (props) => {
  return <motion.button className={twMerge(classNames("rounded-lg border flex justify-center items-center text-xs px-2 py-1 cursor-pointer gap-1 bg-white hover:bg-gray-100", props.className, {
    "cursor-not-allowed": props.disabled,
    "h-[35px] rounded-[4px] gap-2 hover:gap-3": props.type === "lg",
  }))} onClick={props.onClick} disabled={props.disabled} whileTap={{ scale: 0.8 }}>
    <div className={classNames("text-xs text-gray-600", props.labelClassName)}>
      {props.label}
    </div>
    {cloneElement(props.icon, {
      className: classNames("w-4 h-4 stroke-gray-600", props.iconClassName),
    })}
  </motion.button>
}

export const AnimatedButton: FC<IButtonProps> = (props) => {
  return <Button {...props} className={twMerge(classNames("transition-all hover:gap-2", props.className))} />
}