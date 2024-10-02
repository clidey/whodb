import {FC} from "react";
import {InternalPage} from "../../components/page";
import {InternalRoutes} from "../../config/routes";

export const ContactUsPage: FC = () => {
    return <InternalPage routes={[InternalRoutes.ContactUs]}>
        <div className="flex justify-center items-center w-full">
            <iframe
                title={"WhoDB Feedback Form"}
                src="https://docs.google.com/forms/d/e/1FAIpQLSfldEyTbzRdtsFX_6fYtntg9N9s_M7zm8wX8JmrOc98IJPX_A/viewform?embedded=true"
                width="100%" height="1500">Loading…
            </iframe>
        </div>
    </InternalPage>
}