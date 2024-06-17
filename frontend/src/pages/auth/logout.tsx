import { FC, useEffect } from "react";
import { AuthActions } from "../../store/auth";
import { useAppDispatch } from "../../store/hooks";
import { LogoutDocument, LogoutMutation, LogoutMutationVariables } from "../../generated/graphql";
import { useMutation } from "@apollo/client";
import { notify } from "../../store/function";

export const LogoutPage: FC = () => {
  const [logout, ] = useMutation<LogoutMutation, LogoutMutationVariables>(LogoutDocument);

  const dispatch = useAppDispatch();
  useEffect(() => {
    logout({
      onCompleted() {
        dispatch(AuthActions.logout());
      },
      onError() {
        notify("Error logging out", "error");
      }
    })
  }, [dispatch, logout]);

  return <div className="flex grow justify-center items-center">
    <div className="text-md">
      Logging out
    </div>
  </div>
}