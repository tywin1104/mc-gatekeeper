import React from "react";
import MaterialTable from "material-table";
import List from "@material-ui/core/List";
import ListItem from "@material-ui/core/ListItem";
import ListItemIcon from "@material-ui/core/ListItemIcon";
import ListItemText from "@material-ui/core/ListItemText";
import NotesIcon from "@material-ui/icons/Notes";
import WcIcon from "@material-ui/icons/Wc";
import FaceIcon from "@material-ui/icons/Face";
import CommentIcon from "@material-ui/icons/Comment";
import moment from "moment";
import RequestsService from "../../../service/RequestsService";
import i18next from "i18next";
import { CSVLink } from "react-csv";

class Table extends React.Component {
  constructor(props) {
    super(props);
    this.download = this.download.bind(this);
    this.state = {
      dataToDownload: []
    };
  }

  onStatusChange = (event, request, newStatus) => {
    event.preventDefault();
    let requestID = request._id;
    RequestsService.handleStatusChangeByAdmin(
      requestID,
      this.props.config,
      newStatus
    )
      .then(res => {
        if (res.status === 200) {
          this.props.handleChangeRequestStatus({
            requestID: requestID,
            // Mock the updated request here to update the parent state
            request: {
              ...request,
              status: newStatus,
              processedTimestamp: new Date().toISOString(),
              admin: "admin"
            }
          });
        }
      })
      .catch(error => {
        if (error.response) {
          if (error.response.status === 500) {
            alert("Internal Server Error");
          } else if (error.response.status === 401) {
            alert("Login session expired. Please login again");
            this.props.history.push("/login");
          }
        }
      });
  };

  download(data) {
    this.setState({ dataToDownload: data }, () => {
      // click the CSVLink component to trigger the CSV download
      this.csvLink.link.click();
    });
  }

  render() {
    let csvHeaders = [
      { label: "ID", key: "_id" },
      { label: "Username", key: "username" },
      { label: "Email", key: "email" },
      { label: "Age", key: "age" },
      { label: "Application Submitted", key: "timestamp" },
      { label: "Status", key: "status" },
      { label: "Processing Time", key: "processedTimestamp" },
      { label: "Assignees", key: "assignees" },
      { label: "Admin", key: "admin" },
      { label: "Note", key: "note" },
      { label: "Application Info", key: "info.applicationText`" }
    ];
    if (!this.props || this.props.config == null) {
      return <div>Loading...</div>;
    }
    // Create a clone of props so that modifications here will not bubbled up
    let requests = JSON.parse(JSON.stringify(this.props.requests));
    // Transform timestamp and null value to be human-readable
    requests = requests.map(function(item) {
      item.timestamp = moment
        .parseZone(item.timestamp)
        .local()
        .format("MM/DD/YYYY HH:mm");
      // In Golang, time.Time zero value corresponds to a certain timestamp in mongodb
      if (item.processedTimestamp === "0001-01-01T00:00:00Z") {
        item.processedTimestamp = "N/A";
      } else {
        item.processedTimestamp = moment
          .parseZone(item.processedTimestamp)
          .local()
          .format("MM/DD/YYYY HH:mm");
      }
      item.admin = item.admin || "N/A";
      return item;
    });
    return (
      <div>
        <CSVLink
          headers={csvHeaders}
          data={this.state.dataToDownload}
          filename="data.csv"
          className="hidden"
          ref={r => (this.csvLink = r)}
          target="_blank"
        />

        <MaterialTable
          title={i18next.t("Dashboard.Table.AllRequests")}
          columns={[
            { title: i18next.t("Dashboard.Table.ID"), field: "_id" },
            { title: i18next.t("Dashboard.Table.Username"), field: "username" },
            { title: i18next.t("Dashboard.Table.Email"), field: "email" },
            {
              title: i18next.t("Dashboard.Table.Submitted"),
              field: "timestamp"
            },
            { title: i18next.t("Dashboard.Table.Status"), field: "status" },
            {
              title: i18next.t("Dashboard.Table.Processed"),
              field: "processedTimestamp"
            },
            { title: i18next.t("Dashboard.Table.Admin"), field: "admin" },
            {
              title: i18next.t("Dashboard.Table.Assignees"),
              field: "assignees"
            }
          ]}
          data={requests}
          detailPanel={rowData => {
            return (
              <div>
                <List component="nav" aria-label="main mailbox folders">
                  <ListItem button>
                    <ListItemIcon>
                      <WcIcon />
                    </ListItemIcon>
                    <ListItemText primary={rowData.gender} />
                  </ListItem>
                  <ListItem button>
                    <ListItemIcon>
                      <FaceIcon />
                    </ListItemIcon>
                    <ListItemText primary={rowData.age} />
                  </ListItem>
                  <ListItem button>
                    <ListItemIcon>
                      <NotesIcon />{" "}
                      <stong>
                        {i18next.t("Dashboard.Table.ApplicationText")}
                      </stong>
                    </ListItemIcon>
                    <ListItemText primary={rowData.info.applicationText} />
                  </ListItem>
                  {/* Hide note section if the data does not contain it */}
                  <ListItem
                    button
                    style={{ display: rowData.note ? "" : "none" }}
                  >
                    <ListItemIcon>
                      <CommentIcon />{" "}
                      <stong>{i18next.t("Dashboard.Table.Note")}</stong>
                    </ListItemIcon>
                    <ListItemText primary={rowData.note} />
                  </ListItem>
                </List>
              </div>
            );
          }}
          onRowClick={(event, rowData, togglePanel) => togglePanel()}
          actions={[
            rowData => ({
              icon: "check",
              tooltip: i18next.t("Dashboard.Table.ApproveTooltip"),
              onClick: (event, rowData) =>
                this.onStatusChange(event, rowData, "Approved"),
              hidden: rowData.status !== "Pending"
            }),
            rowData => ({
              icon: "close",
              tooltip: i18next.t("Dashboard.Table.DenyTooltip"),
              onClick: (event, rowData) =>
                this.onStatusChange(event, rowData, "Denied"),
              hidden: rowData.status !== "Pending"
            }),
            rowData => ({
              icon: "cancel",
              tooltip: "Deactivate the user",
              onClick: (event, rowData) =>
                this.onStatusChange(event, rowData, "Deactivated"),
              hidden: rowData.status !== "Approved"
            }),
            rowData => ({
              icon: "delete",
              tooltip: "Ban the user",
              onClick: (event, rowData) =>
                this.onStatusChange(event, rowData, "Banned"),
              hidden: rowData.status !== "Approved"
            })
          ]}
          options={{
            actionsColumnIndex: -1,
            exportButton: true,
            exportCsv: (columns, data) => {
              this.download(data);
            },
            grouping: true
          }}
        />
      </div>
    );
  }
}

export default Table;
