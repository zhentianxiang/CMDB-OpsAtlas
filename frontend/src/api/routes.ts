type Result = {
  code: number;
  message: string;
  data: Array<any>;
};

export const getAsyncRoutes = () => {
  return Promise.resolve<Result>({
    code: 0,
    message: "success",
    data: []
  });
};
