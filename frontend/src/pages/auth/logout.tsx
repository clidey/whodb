import { useMutation } from "@apollo/client";
import { FC, useEffect } from "react";
import { useDispatch } from "react-redux";
import { Icons } from "../../components/icons";
import { Page } from "../../components/page";
import { LogoutDocument, LogoutMutation, LogoutMutationVariables } from "../../generated/graphql";
import { AuthActions } from "../../store/auth";
import { notify } from "../../store/function";

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

  return <Page className="text-neutral-800 dark:text-neutral-300 flex justify-center items-center">
    {Icons.Lock}
    <div className="text-md">
      Logging out
    </div>
  </Page>
}