import { useMutation } from "@apollo/client";
import { FC, cloneElement, useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { AnimatedButton } from "../../components/button";
import { BASE_CARD_CLASS, BRAND_COLOR } from "../../components/classes";
import { DropdownWithLabel, IDropdownItem } from "../../components/dropdown";
import { Icons } from "../../components/icons";
import { InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { Page } from "../../components/page";
import { LoginDocument, LoginMutation, LoginMutationVariables } from '../../generated/graphql';
import { AuthActions } from "../../store/auth";
import { notify } from "../../store/function";
import { useAppDispatch } from "../../store/hooks";
import { isNumeric } from "../../utils/functions";
import { InternalRoutes } from "../../config/routes";

const databaseDropdownItems: IDropdownItem[] = [
    {
        id: "Postgres",
        label: "Postgres",
        icon: Icons.Logos.Postgres,
    },
    {
        id: "MySQL",
        label: "MySQL",
        icon: Icons.Logos.MySQL,
    },
]

export const LoginPage: FC = () => {
    const dispatch = useAppDispatch();
    const navigate = useNavigate();
    
    const [login, { loading }] = useMutation<LoginMutation, LoginMutationVariables>(LoginDocument);

    const [databaseType, setDatabaseType] = useState<IDropdownItem>(databaseDropdownItems[0]);
    const [hostName, setHostName] = useState("");
    const [port, setPort] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string>();

    const handleSubmit = useCallback(() => {
        if (hostName.length === 0 || port.length === 0 || username.length === 0 || password.length === 0) {
            return setError(`All fields are required`);
        }
        if (!isNumeric(port)) {
            return setError("Port has to be a number")
        }
        setError(undefined);

        const credentails = {
            Type: databaseType.id,
            Hostname: hostName,
            Port: Number.parseInt(port),
            Username: username,
            Password: password,
        };

        login({
            variables: {
                credentails,
            },
            onCompleted(data) {
                if (data.Login.Status) {
                    dispatch(AuthActions.login(credentails));
                    navigate(InternalRoutes.Dashboard.path);
                    return notify("Login successfully", "success");
                }
                return notify("Login failed", "error");
            },
            onError(error) {
                return notify(`Login failed: ${error.message}`, "error");
            }
        })
    }, [databaseType.id, dispatch, hostName, login, navigate, password, port, username]);

    if (loading)  {
        return (
            <Page className="justify-center items-center">
                <div className={twMerge(BASE_CARD_CLASS, "w-[350px] h-fit flex-col gap-2 justify-center")}>
                    <Loading />
                    <div className="text-gray-600">
                        Logging in
                    </div>
                </div>
            </Page>)
    }

    return (
        <Page className="justify-center items-center">
            <div className={twMerge(BASE_CARD_CLASS, "w-[350px] h-fit")}>
                <div className="flex flex-col justify-between grow gap-4">
                    <div className="flex flex-col gap-4 grow">
                        <div className="flex justify-between">
                            <div className="text-lg text-gray-600 flex gap-2 items-center">
                                <div className="h-[40px] w-[40px] rounded-xl flex justify-center items-center bg-teal-500">
                                    {cloneElement(Icons.Lock, {
                                        className: "w-6 h-6 stroke-white",
                                    })}
                                </div>
                                <span className={BRAND_COLOR}>WhoDB</span> Login
                            </div>
                            <div className="text-red-500 text-xs flex items-center">
                                {error}
                            </div>
                        </div>
                        <div className="flex flex-col grow justify-center gap-1">
                            <DropdownWithLabel label="Database" value={databaseType} onChange={setDatabaseType} items={databaseDropdownItems} />
                            <InputWithlabel label="Host Name" value={hostName} setValue={setHostName} />
                            <InputWithlabel label="Port" value={port} setValue={setPort} />
                            <InputWithlabel label="Username" value={username} setValue={setUsername} />
                            <InputWithlabel label="Password" value={password} setValue={setPassword} type="password" />
                        </div>
                    </div>
                    <div className="flex justify-end">
                        <AnimatedButton icon={Icons.CheckCircle} label="Submit" onClick={handleSubmit} />
                    </div>
                </div>
            </div>
        </Page>
    )
}