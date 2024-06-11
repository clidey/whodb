import { FC, cloneElement, useState } from "react";
import { twMerge } from "tailwind-merge";
import { BASE_CARD_CLASS, BRAND_COLOR } from "../components/classes";
import { Page } from "../components/page";
import { InputWithlabel } from "../components/input";
import { AnimatedButton } from "../components/button";
import { Icons } from "../components/icons";


export const LoginPage: FC = () => {
    const [hostName, setHostName] = useState("");
    const [port, setPort] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");

    return (
        <Page className="justify-center items-center">
            <div className={twMerge(BASE_CARD_CLASS, "w-[350px] h-[400px]")}>
                <div className="flex flex-col justify-between grow">
                    <div className="flex flex-col gap-4 grow">
                        <div className="text-2xl text-gray-600 flex gap-2 items-center">
                            <div className="h-[50px] w-[50px] rounded-xl flex justify-center items-center bg-teal-500">
                                {cloneElement(Icons.Lock, {
                                    className: "w-6 h-6 stroke-white",
                                })}
                            </div>
                            <span className={BRAND_COLOR}>WhoDB</span> Login
                        </div>
                        <div className="flex flex-col grow justify-center gap-1">
                            <InputWithlabel label="Host Name" value={hostName} setValue={setHostName} />
                            <InputWithlabel label="Port" value={port} setValue={setPort} />
                            <InputWithlabel label="Username" value={username} setValue={setUsername} />
                            <InputWithlabel label="Password" value={password} setValue={setPassword} type="password" />
                        </div>
                    </div>
                    <div className="flex justify-end">
                        <AnimatedButton icon={Icons.CheckCircle} label="Submit" />
                    </div>
                </div>
            </div>
        </Page>
    )
}