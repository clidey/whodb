import { FC, useEffect } from "react";
import { AuthActions } from "../store/auth";
import { useAppDispatch } from "../store/hooks";

export const LogoutPage: FC = () => {
  const dispatch = useAppDispatch();
  useEffect(() => {
    dispatch(AuthActions.logout());
  }, [dispatch]);

  return <div className="flex grow justify-center items-center">
    <div className="text-md">
      Logging out
    </div>
  </div>
}