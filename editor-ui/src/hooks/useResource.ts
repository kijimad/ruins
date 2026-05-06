import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";

// 汎用リソース CRUD hooks
// リソースURLとクエリキーを渡すと list/get/create/update/delete を提供する

interface ListResponse<T> {
  data: T[];
  totalCount: number;
}

export function useResourceList<T>(resource: string) {
  return useQuery<ListResponse<T>>({
    queryKey: [resource],
    queryFn: async () => {
      const res = await axios.get<ListResponse<T>>(`/api/v1/${resource}`);
      return res.data;
    },
  });
}

export function useResourceUpdate<T>(resource: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ index, data }: { index: number; data: T }) => {
      const res = await axios.put<T>(`/api/v1/${resource}/${index}`, data);
      return res.data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [resource] });
    },
  });
}

interface CreateResponse<T> {
  index: number;
  data: T;
}

export function useResourceCreate<T>(resource: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (data: T) => {
      const res = await axios.post<CreateResponse<T>>(`/api/v1/${resource}`, data);
      return res.data;
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: [resource] });
    },
  });
}

export function useResourceDelete(resource: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (index: number) => {
      await axios.delete(`/api/v1/${resource}/${index}`);
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [resource] });
    },
  });
}
