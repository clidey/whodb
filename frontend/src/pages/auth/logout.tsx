import { useMutation } from "@apollo/client";
import { FC, useEffect } from "react";
import { useDispatch } from "react-redux";
import { LogoutDocument, LogoutMutation, LogoutMutationVariables } from "../../generated/graphql";
import { notify } from "../../store/function";
import { AuthActions } from "../../store/auth";

export const LogoutPage: FC = () => {
  const [logout, ] = useMutation<LogoutMutation, LogoutMutationVariables>(LogoutDocument);
  const dispatch = useDispatch();

  useEffect(() => {
    logout({
      onCompleted() {
        dispatch(AuthActions.logout());
        notify("Logged out successfully", "success");
      },
      onError() {
        notify("Error logging out", "error");
      }
    });
  }, [dispatch, logout]);

  return <div className="flex grow justify-center items-center">
    <div className="text-md">
      Logging out
    </div>
  </div>
}