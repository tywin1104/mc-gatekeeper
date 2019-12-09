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
import Button from "@material-ui/core/Button";
import Dialog from "@material-ui/core/Dialog";
import DialogActions from "@material-ui/core/DialogActions";
import DialogContent from "@material-ui/core/DialogContent";
import DialogContentText from "@material-ui/core/DialogContentText";
import DialogTitle from "@material-ui/core/DialogTitle";
import { CSVLink } from "react-csv";

class Table extends React.Component {
  constructor(props) {
    super(props);
    this.download = this.download.bind(this);
    this.state = {
      open: false,
      dataToDownload: []
    };
  }

  handleClickOpen = () => {
    this.setState({ open: true });
  };

  handleClose = () => {
    this.setState({ open: false });
  };
  onStatusChange = (request, newStatus) => {
    let requestID = request._id;
    RequestsService.handleStatusChangeByAdmin(
      requestID,
      this.props.config,
      newStatus
    )
      .then(res => {
        if (res.status === 200) {
          let processedTimestamp = request.processedTimestamp;
          if (processedTimestamp === "N/A") {
            processedTimestamp = new Date().toISOString();
          }
          this.props.handleChangeRequestStatus({
            requestID: requestID,
            // Mock the updated request here to update the parent state
            request: {
              ...request,
              status: newStatus,
              processedTimestamp: processedTimestamp,
              lastUpdatedTimestamp: new Date().toISOString()
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
          } else {
            alert(
              "Unable to complete the request. Please refresh and try again"
            );
          }
        }
      });
  };

  // onAttempt will open up the confirmation dialog for each corresponding actions
  onAttemptAction = (rowData, newStatus) => {
    this.setState({
      open: true,
      rowData: rowData,
      attemptedNewStatus: newStatus
    });
  };

  onConfirmAction = () => {
    this.onStatusChange(this.state.rowData, this.state.attemptedNewStatus);
    this.setState({ open: false });
  };

  getActionConfirmMsg = () => {
    let attemptedNewStatus = this.state.attemptedNewStatus;
    if (attemptedNewStatus === "Banned") {
      return "You are about to ban the player permanately on your server. Are you sure about this?";
    } else if (attemptedNewStatus === "Deactivated") {
      return "By deactivating, the player will be unwhitelisted from your server and unable to play. However the user will be able to submit new application again in the future.";
    }
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
      { label: "Processed Time", key: "processedTimestamp" },
      { label: "Last Status Update Time", key: "lastUpdatedTimestamp" },
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
      if (item.lastUpdatedTimestamp === "0001-01-01T00:00:00Z") {
        item.lastUpdatedTimestamp = "N/A";
      } else {
        item.lastUpdatedTimestamp = moment
          .parseZone(item.lastUpdatedTimestamp)
          .local()
          .format("MM/DD/YYYY HH:mm");
      }
      item.admin = item.admin || "N/A";
      return item;
    });
    return (
      <div>
        <div>
          <Dialog
            open={this.state.open}
            onClose={this.handleClose}
            aria-labelledby="alert-dialog-title"
            aria-describedby="alert-dialog-description"
          >
            <DialogTitle id="alert-dialog-title">{"Confirmation"}</DialogTitle>
            <DialogContent>
              <DialogContentText id="alert-dialog-description">
                {this.getActionConfirmMsg()}
              </DialogContentText>
            </DialogContent>
            <DialogActions>
              <Button onClick={this.handleClose} color="primary">
                Cancel
              </Button>
              <Button onClick={this.onConfirmAction} color="primary" autoFocus>
                Confirm
              </Button>
            </DialogActions>
          </Dialog>
        </div>
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
            { title: "Gender", field: "gender" },
            { title: "Age", field: "age" },
            {
              title: i18next.t("Dashboard.Table.Submitted"),
              field: "timestamp"
            },
            { title: i18next.t("Dashboard.Table.Status"), field: "status" },
            {
              title: i18next.t("Dashboard.Table.Processed"),
              field: "processedTimestamp"
            },
            {
              title: i18next.t("Dashboard.Table.LastUpdatedTimestamp"),
              field: "lastUpdatedTimestamp"
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
                  <ListItem>
                    <ListItemIcon>
                      <NotesIcon />{" "}
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
                this.onStatusChange(rowData, "Approved"),
              hidden: rowData.status !== "Pending"
            }),
            rowData => ({
              icon: "close",
              tooltip: i18next.t("Dashboard.Table.DenyTooltip"),
              onClick: (event, rowData) =>
                this.onStatusChange(rowData, "Denied"),
              hidden: rowData.status !== "Pending"
            }),
            rowData => ({
              icon: "cancel",
              tooltip: "Deactivate the user",
              onClick: (event, rowData) =>
                this.onAttemptAction(rowData, "Deactivated"),
              hidden: rowData.status !== "Approved"
            }),
            rowData => ({
              icon: "delete",
              tooltip: "Ban the user",
              onClick: (event, rowData) =>
                this.onAttemptAction(rowData, "Banned"),
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
