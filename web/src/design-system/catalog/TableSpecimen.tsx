import { useState } from "react";
import { Table, TableHead, TableCell } from "../components/table";
import { Pagination } from "../components/pagination";
import { Input } from "../components/input";
import { Field, FieldLabel } from "../components/field";
import { Empty } from "../components/empty";
import { Button } from "../components/button";
import { Badge } from "../components/badge";
import type { CatalogLocale } from "./catalog-locale";

const rows = [
  {
    id: "r1",
    asset: "RUB",
    amount: "125000.00",
    ru: "Продукты",
    en: "Groceries",
  },
  { id: "r2", asset: "USD", amount: "1820.75", ru: "Дом", en: "Home" },
  {
    id: "r3",
    asset: "USDT",
    amount: "400.00000001",
    ru: "Накопление",
    en: "Savings",
  },
  {
    id: "r4",
    asset: "USDC",
    amount: "0.123456789012345678",
    ru: "Пополнение",
    en: "Deposit",
  },
  { id: "r5", asset: "BTC", amount: "0.12345678", ru: "Комиссия", en: "Fee" },
  {
    id: "r6",
    asset: "ETH",
    amount: "0.123456789012345678",
    ru: "Возврат",
    en: "Refund",
  },
];
export function TableSpecimen({ locale }: { locale: CatalogLocale }) {
  const ru = locale === "ru";
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(0);
  const filtered = rows.filter((row) =>
    `${row[locale]} ${row.asset}`
      .toLocaleLowerCase()
      .includes(query.toLocaleLowerCase()),
  );
  const visible = filtered.slice(page * 3, page * 3 + 3);
  return (
    <div className="catalog-stack">
      <div className="catalog-heading">
        <p className="catalog-eyebrow">04 / {ru ? "Таблицы" : "Tables"}</p>
        <h2>
          {ru ? "Суммы, которые можно прочитать" : "Every digit stays readable"}
        </h2>
      </div>
      <Field>
        <FieldLabel>
          {ru ? "Найти запись или актив" : "Find an entry or asset"}
        </FieldLabel>
        <Input
          value={query}
          onValueChange={(next) => {
            setQuery(next);
            setPage(0);
          }}
        />
      </Field>
      {visible.length ? (
        <Table label={ru ? "Пример точных сумм" : "Exact amounts sample"}>
          <caption>
            {ru
              ? "Синтетические записи; валюты не суммируются"
              : "Synthetic entries; currencies are not summed"}
          </caption>
          <thead>
            <tr>
              <TableHead>{ru ? "Назначение" : "Purpose"}</TableHead>
              <TableHead>{ru ? "Сумма" : "Amount"}</TableHead>
              <TableHead>{ru ? "Актив" : "Asset"}</TableHead>
              <TableHead>{ru ? "Состояние" : "Status"}</TableHead>
            </tr>
          </thead>
          <tbody>
            {visible.map((row) => (
              <tr key={row.id}>
                <TableCell>{row[locale]}</TableCell>
                <TableCell
                  className="wk-exact-amount"
                  data-testid={`table-${row.asset}`}
                >
                  {row.amount}
                </TableCell>
                <TableCell>{row.asset}</TableCell>
                <TableCell>
                  <Badge>{ru ? "Образец" : "Sample"}</Badge>
                </TableCell>
              </tr>
            ))}
          </tbody>
        </Table>
      ) : (
        <Empty
          title={ru ? "Нет совпадений" : "No matches"}
          description={
            ru
              ? "Измените запрос или сбросьте фильтр."
              : "Change the query or clear the filter."
          }
          action={
            <Button variant="secondary" onClick={() => setQuery("")}>
              {ru ? "Сбросить фильтр" : "Clear filter"}
            </Button>
          }
        />
      )}
      <Pagination
        label={ru ? "Страницы записей" : "Entry pages"}
        previousLabel={ru ? "Назад" : "Previous"}
        nextLabel={ru ? "Далее" : "Next"}
        summary={`${page + 1} / ${Math.max(1, Math.ceil(filtered.length / 3))}`}
        hasPrevious={page > 0}
        hasNext={(page + 1) * 3 < filtered.length}
        onPrevious={() => setPage((value) => value - 1)}
        onNext={() => setPage((value) => value + 1)}
      />
    </div>
  );
}
