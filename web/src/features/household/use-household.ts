import { useContext } from "react";
import { HouseholdContext } from "./household-context";

export function useHousehold() {
  const context = useContext(HouseholdContext);
  if (!context) throw new Error("Household provider is required");
  return context;
}
