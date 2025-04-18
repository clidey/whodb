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

import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { FC, cloneElement, useCallback, useEffect } from "react";
import { CommonActions, INotification } from "../store/common";
import { useAppDispatch, useAppSelector } from "../store/hooks";
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
    <div className="relative px-4 py-2 text-sm grid grid-col-[1fr_auto] gap-2 overflow-hidden h-auto">
      <p className="dark:text-neutral-300">{notification.message}</p>
      <div className="flex justify-end items-center">
        <button
          type="button"
          className="z-[100] rounded-full transition-all hover:scale-110"
          onClick={handleRemove}
        >
          {cloneElement(Icons.Cancel, {
            className: "w-6 h-6 stroke-neutral-800 dark:stroke-neutral-500",
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
    <div className="fixed z-[100] w-auto top-8 bottom-8 m-[0_auto] left-8 right-8 flex flex-col gap-2 items-end justify-end xs:items-center pointer-events-none">
      <AnimatePresence mode="sync">
        <motion.ul className="flex flex-col gap-4">
          {notifications.map((notification) => (
            <motion.li
              key={notification.id}
              layout
              className={classNames("bg-white bg-white/10 backdrop-blur-lg box-border overflow-hidden w-[30ch] sm:width-full shadow-lg rounded-2xl border border-neutral-200 dark:border-neutral-200/5 pointer-events-auto border-r-[20px]", {
                  "border-r-neutral-400 dark:border-r-neutral-600": notification.intent === "default",
                  "border-r-red-400 dark:border-r-red-600": notification.intent === "error",
                  "border-r-orange-400 dark:border-r-orange-600": notification.intent === "warning",
                  "border-r-green-400 dark:border-r-green-600": notification.intent === "success",
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
