import createMiddleware from "next-intl/middleware";
import { routing } from "./i18n/routing";

export default createMiddleware(routing);

export const config = {
  matcher: ["/", "/(en|zh|ja|ko)/:path*", "/((?!api|_next|_vercel|_next/static|_next/image|favicon.ico|.*\\..*).*)"],
};
