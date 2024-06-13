import { v4 } from "uuid"
import { reduxStore } from "."
import { CommonActions, IIntent } from "./common"

export function notify(message: string, intent: IIntent = "default") {
    reduxStore.dispatch(CommonActions.addNotifications({
        id: v4(),
        message,
        intent,
    }));
}