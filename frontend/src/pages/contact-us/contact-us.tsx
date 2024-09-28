import {FC} from "react";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";

export const ContactUsPage: FC = () => {
    return <InternalPage routes={[InternalRoutes.ContactUs]}>
        <div className="flex justify-center items-center w-full">
            <iframe
                title={"WhoDB Feedback Form"}
                src="" // todo: add your own google forms link here
                width="100%" height="1500">Loading…
            </iframe>
        </div>
    </InternalPage>
}