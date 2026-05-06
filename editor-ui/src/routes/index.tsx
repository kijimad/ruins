import { createBrowserRouter } from "react-router-dom";
import { Layout, WelcomePage } from "../Layout";
import { ResourcePage } from "../pages/ResourcePage";

export const router = createBrowserRouter([
  {
    element: <Layout />,
    children: [
      { index: true, element: <WelcomePage /> },
      { path: "items", element: <ResourcePage resource="items" label="アイテム" /> },
      { path: "members", element: <ResourcePage resource="members" label="メンバー" /> },
      { path: "recipes", element: <ResourcePage resource="recipes" label="レシピ" /> },
      { path: "command-tables", element: <ResourcePage resource="command-tables" label="コマンドテーブル" /> },
      { path: "drop-tables", element: <ResourcePage resource="drop-tables" label="ドロップテーブル" /> },
      { path: "item-tables", element: <ResourcePage resource="item-tables" label="アイテムテーブル" /> },
      { path: "enemy-tables", element: <ResourcePage resource="enemy-tables" label="敵テーブル" /> },
      { path: "tiles", element: <ResourcePage resource="tiles" label="タイル" /> },
      { path: "props", element: <ResourcePage resource="props" label="置物" /> },
      { path: "professions", element: <ResourcePage resource="professions" label="職業" nameField="id" /> },
      { path: "sprite-sheets", element: <ResourcePage resource="sprite-sheets" label="スプライトシート" /> },
      { path: "palettes", element: <ResourcePage resource="palettes" label="パレット" nameField="id" /> },
    ],
  },
]);
