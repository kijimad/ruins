import { Configuration, DefaultApi } from "../oapi";

export const config = new Configuration({
  basePath: "",
});

export const apiClient = new DefaultApi(config);
