import { AnimatePresence, motion } from "framer-motion";
import { FC, cloneElement, useCallback, useEffect } from "react";
import { CommonActions, INotification } from "../store/common";
import { useAppDispatch, useAppSelector } from "../store/hooks";
import classNames from "classnames";
import { Icons } from "./icons";

type INotificationProps = {
  notification: INotification;
};

const Notification: FC<INotificationProps> = ({ notification }) => {
  const dispatch = useAppDispatch();

  const handleRemove = useCallback(() => {
    dispatch(CommonActions.removeNotifications(notification));
  }, [dispatch, notification]);

  useEffect(() => {
    const timeout = setTimeout(() => {
      dispatch(CommonActions.removeNotifications(notification));
    }, 5000);
    return () => {
      clearTimeout(timeout);
    }
  }, [dispatch, notification]);

  return (
    <div className="relative px-4 py-6 text-sm grid grid-col-[1fr_auto] gap-2 overflow-hidden h-auto">
      <p>{notification.message}</p>
      <div className="flex justify-end items-center">
        <button
          type="button"
          className="z-100 rounded-full transition-all hover:scale-110"
          onClick={handleRemove}
        >
          {cloneElement(Icons.Cancel, {
            className: "w-6 h-6 stroke-green-800",
          })}
        </button>
      </div>
    </div>
  );
};

type INotificationsProps = {};

export const Notifications: FC<INotificationsProps> = () => {
  const notifications = useAppSelector((state) => state.common.notifications);

  return (
    <div className="fixed z-50 w-auto top-8 bottom-8 m-[0_auto] left-8 right-8 flex flex-col gap-2 items-end xs:items-center pointer-events-none">
      <AnimatePresence mode="sync">
        <motion.ul className="flex flex-col gap-4">
          {notifications.map((notification) => (
            <motion.li
              key={notification.id}
              layout
              className={classNames("bg-white box-border overflow-hidden w-[40ch] sm:width-full shadow-lg rounded-xl border border-gray-200 pointer-events-auto border-r-8", {
                  "border-r-gray-400": notification.intent === "default",
                  "border-r-red-400": notification.intent === "error",
                  "border-r-orange-400": notification.intent === "warning",
                  "border-r-green-400": notification.intent === "success",
              })}
              initial={{ opacity: 0, y: 50, scale: 0.3 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, scale: 0.5, transition: { duration: 0.2 } }}
            >
              <Notification notification={notification} />
            </motion.li>
          ))}
        </motion.ul>
      </AnimatePresence>
    </div>
  );
};
