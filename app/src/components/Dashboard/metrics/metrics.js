import React from "react";
import { withStyles } from "@material-ui/core/styles";
import Container from "@material-ui/core/Container";
import Grid from "@material-ui/core/Grid";
import Card from "@material-ui/core/Card";
import CardContent from "@material-ui/core/CardContent";
import Typography from "@material-ui/core/Typography";
import RequestsService from "../../../service/RequestsService";
import {
  ResponsiveContainer,
  RadialBarChart,
  RadialBar,
  Legend,
  FunnelChart,
  Funnel,
  LabelList,
  Tooltip,
  PieChart,
  Pie,
  Label
} from "recharts";

const useStyles = theme => ({
  root: {
    marginTop: "5%",
    flexGrow: 1
  }
});

class Metrics extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      source: RequestsService.getStatsEventSource(),
      stats: {}
    };
  }
  componentDidMount() {
    const { source } = this.state;
    source.addEventListener("closedConnection", e => this.source.close());
    // Update the stats once data arrived from the sever via SSE
    source.addEventListener("message", message => {
      // console.log(message);
      this.updateStats(message.data);
    });
  }

  updateStats = data => {
    this.setState({ stats: JSON.parse(data) });
  };

  getStatusGroupData = () => {};

  render() {
    let statusGroupData = [
      {
        value: this.state.stats.pending,
        name: "Pending",
        fill: "#8884d8"
      },
      {
        value: this.state.stats.approved,
        name: "Approved",
        fill: "#83a6ed"
      },
      {
        value: this.state.stats.denied,
        name: "Denied",
        fill: "#8dd1e1"
      }
    ];
    let genderGroupData = [
      {
        name: "Male",
        value: this.state.stats.maleCount
      },
      {
        name: "Female",
        value: this.state.stats.femaleCount
      },
      {
        name: "Others",
        value: this.state.stats.otherGenderCount
      }
    ];
    let ageGroupData = [
      {
        name: "0-15",
        value: this.state.stats.ageGroup1Count,
        fill: "#8884d8"
      },
      {
        name: "15-30",
        value: this.state.stats.ageGroup2Count,
        fill: "#83a6ed"
      },
      {
        name: "30-45",
        value: this.state.stats.ageGroup3Count,
        fill: "#8dd1e1"
      },
      {
        name: "45+",
        value: this.state.stats.ageGroup4Count,
        fill: "#82ca9d"
      }
    ];
    const { classes } = this.props;
    return (
      <Container maxWidth="lg" className={classes.root}>
        <Grid
          container
          direction="row"
          justify="flex-start"
          alignItems="flex-start"
          spacing={3}
        >
          <Grid item xs={12} sm={3}>
            <Card className={classes.card}>
              <CardContent>
                <Typography component="h2">Pending Requests Count</Typography>
                <Typography component="p" variant="h4">
                  {this.state.stats.pending}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={3}>
            <Card className={classes.card}>
              <CardContent>
                <Typography component="h2">Approved Requests Count</Typography>
                <Typography component="p" variant="h4">
                  {this.state.stats.approved}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={3}>
            <Card className={classes.card}>
              <CardContent>
                <Typography component="h2">Denied Requests Count</Typography>
                <Typography component="p" variant="h4">
                  {this.state.stats.denied}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={3}>
            <Card className={classes.card}>
              <CardContent>
                <Typography component="h2">Average Response Time</Typography>
                <Typography component="p" variant="h4">
                  {this.state.stats.averageResponseTimeInMinutes} Minutes
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={12}>
            <ResponsiveContainer width="80%" height={300}>
              <FunnelChart>
                <Tooltip />
                <Funnel
                  dataKey="value"
                  data={statusGroupData}
                  isAnimationActive
                >
                  <LabelList
                    position="right"
                    fill="#000"
                    stroke="none"
                    dataKey="name"
                  />
                </Funnel>
              </FunnelChart>
            </ResponsiveContainer>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography component="p" variant="h6" align="center">
              Gender Distribution for Approved Applications
            </Typography>
            <ResponsiveContainer width="100%" height={400}>
              <PieChart>
                <Pie
                  data={genderGroupData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  innerRadius={30}
                  outerRadius={90}
                  fill="#82ca9d"
                  label={({
                    cx,
                    cy,
                    midAngle,
                    innerRadius,
                    outerRadius,
                    value,
                    index
                  }) => {
                    console.log("handling label?");
                    const RADIAN = Math.PI / 180;
                    // eslint-disable-next-line
                    const radius =
                      25 + innerRadius + (outerRadius - innerRadius);
                    // eslint-disable-next-line
                    const x = cx + radius * Math.cos(-midAngle * RADIAN);
                    // eslint-disable-next-line
                    const y = cy + radius * Math.sin(-midAngle * RADIAN);

                    return (
                      <text
                        x={x}
                        y={y}
                        fill="#111111"
                        textAnchor={x > cx ? "start" : "end"}
                        dominantBaseline="central"
                      >
                        {genderGroupData[index].name} ({value})
                      </text>
                    );
                  }}
                />
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography component="p" variant="h6" align="center">
              Age Distribution for Approved Applications
            </Typography>
            <ResponsiveContainer width="100%" height={500}>
              <RadialBarChart
                innerRadius="10%"
                outerRadius="80%"
                data={ageGroupData}
                startAngle={180}
                endAngle={0}
              >
                <RadialBar
                  minAngle={15}
                  background
                  clockWise={true}
                  dataKey="value"
                />
                <Legend
                  iconSize={10}
                  layout="vertical"
                  verticalAlign="top"
                  align="right"
                />
                <Tooltip />
              </RadialBarChart>
            </ResponsiveContainer>
          </Grid>
        </Grid>
      </Container>
    );
  }
}

export default withStyles(useStyles)(Metrics);
