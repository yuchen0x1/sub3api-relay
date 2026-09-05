import { calculateCapacity } from "../capacity"

describe("calculateCapacity", () => {
  it("calculates remaining capacity and utilization", () => {
    expect(calculateCapacity(3, 10)).toEqual({ used: 3, limit: 10, available: 7, utilization: 0.3 })
  })

  it("clamps available capacity when usage exceeds the limit", () => {
    expect(calculateCapacity("12", "10")).toEqual({ used: 12, limit: 10, available: 0, utilization: 1.2 })
  })

  it("represents an absent limit as unlimited", () => {
    expect(calculateCapacity(4, undefined)).toEqual({ used: 4, limit: null, available: null, utilization: null })
  })

  it("handles a zero limit without dividing by zero", () => {
    expect(calculateCapacity(0, 0)).toEqual({ used: 0, limit: 0, available: 0, utilization: 0 })
  })
})
