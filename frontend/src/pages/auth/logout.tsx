import { useMutation } from "@apollo/client";
import { FC, useEffect } from "react";
import { useDispatch } from "react-redux";
import { Icons } from "../../components/icons";
import { Container, Page } from "../../components/page";
import { LogoutDocument, LogoutMutation, LogoutMutationVariables } from "../../generated/graphql";
import { AuthActions } from "../../store/auth";
import { notify } from "../../store/function";
import { Loading } from "../../components/loading";

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

  return <Container>
      <div className="flex flex-col justify-center items-center gap-4 w-full">
          <div>
              <Loading hideText={true} />
          </div>
          <div className="text-neutral-800 dark:text-neutral-300">
              Logging out
          </div>
      </div>
  </Container>
}