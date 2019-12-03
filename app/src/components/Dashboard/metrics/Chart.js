import React from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Label,
  ResponsiveContainer,
  Tooltip
} from "recharts";
import moment from "moment";
import i18next from "i18next";
let _ = require("lodash");

function createData(time, amount) {
  return { time, amount };
}

export default function Chart(props) {
  let data = [];
  // Create a clone of props so that modifications here will not bubbled up
  let requests = JSON.parse(JSON.stringify(props.requests));
  // Group all requests by date -> [request]
  let groupedResults = _.groupBy(requests, request =>
    moment
      .parseZone(request["timestamp"])
      .local()
      .format("MM/DD/YYYY")
  );
  let daysAgo = [];
  for (let i = 4; i >= 0; i--) {
    daysAgo.push(
      moment()
        .subtract(i, "days")
        .format("MM/DD/YYYY")
    );
  }
  // Append data for last 5 days in order to display in chart
  for (let i = 0; i < daysAgo.length; i++) {
    let date = daysAgo[i];
    if (Object.prototype.hasOwnProperty.call(groupedResults, date)) {
      data.push(createData(date, groupedResults[date].length));
    } else {
      // If no new requests for today, set graph line height to zero
      data.push(createData(date, 0));
    }
  }

  return (
    <React.Fragment>
      {i18next.t("Dashboard.Chart.Title")}
      <ResponsiveContainer>
        <LineChart
          data={data}
          margin={{
            top: 16,
            right: 16,
            bottom: 0,
            left: 24
          }}
        >
          <XAxis dataKey="time" />
          <YAxis>
            <Label angle={270} position="left" style={{ textAnchor: "middle" }}>
              {i18next.t("Dashboard.Chart.Count")}
            </Label>
          </YAxis>
          <Tooltip />
          <Line type="monotone" dataKey="amount" stroke="#556CD6" dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </React.Fragment>
  );
}
