import React from "react";
import axios from "axios";
import MaterialTable from "material-table";
import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import ListItemText from '@material-ui/core/ListItemText';
import InboxIcon from '@material-ui/icons/Inbox';
import DraftsIcon from '@material-ui/icons/Drafts';
import WcIcon from '@material-ui/icons/Wc';
import FaceIcon from '@material-ui/icons/Face';
import moment from 'moment'

class Table extends React.Component {
  onApprove = (event, request) => {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    let requestID = request._id
    event.preventDefault();
    axios.patch(`${api_base_url}/api/v1/internal/requests/${requestID}`, {
        status: "Approved",
        processedTimestamp: new Date().toISOString(),
        admin: 'admin'
    })
    .then(res => {
      if (res.status === 200) {
          window.location.reload();
      }})
    .catch(error => {
       alert("Internal server error")
    });
  }

  onDeny = (event, request) => {
    let api_base_url = process.env.REACT_APP_API_BASE_URL
    let requestID = request._id
    event.preventDefault();
    axios.patch(`${api_base_url}/api/v1/internal/requests/${requestID}`, {
        status: "Denied",
        processedTimestamp: new Date().toISOString(),
        admin: 'admin'
    })
    .then(res => {
      if (res.status === 200) {
        window.location.reload();
      }})
    .catch(error => {
       alert("Internal server error")
    });
  }


  render() {
    if(!this.props || this.props.requests.length === 0) {
      return <div>Loading...</div>
    }
    // Create a clone of props so that modifications here will not bubbled up
    let requests = JSON.parse(JSON.stringify(this.props.requests))
    // Transform timestamp and null value to be human-readable
    requests = requests.map(function(item){
      item.timestamp =  moment.parseZone(item.timestamp).local().format("MM/DD/YYYY HH:mm")
      console.log(item.processedTimestamp)
      // In Golang, time.Time zero value corresponds to a certain timestamp in mongodb
      if(item.processedTimestamp === "0001-01-01T00:00:00Z") {
        item.processedTimestamp = "N/A"
      }else {
        item.processedTimestamp =  moment.parseZone(item.processedTimestamp).local().format("MM/DD/YYYY HH:mm")
      }
      item.admin = item.admin || "N/A"
      return item;
    });
    return (
      <MaterialTable
        title="View all requests"
        columns={[
          { title: 'Username', field: 'username' },
          { title: 'Email', field: 'email' },
          { title: 'Application Submitted', field: 'timestamp', },
          { title: 'Status', field: 'status',},
          { title: 'Processed', field: 'processedTimestamp',},
          { title: 'Admin', field: 'admin',},
        ]}
        data = {requests}
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
                  <DraftsIcon />
                </ListItemIcon>
                <ListItemText primary={rowData.info.applicationText}/>
              </ListItem>
            </List>
            </div>
          )
        }}
        onRowClick={(event, rowData, togglePanel) => togglePanel()}
        actions={[
          rowData => ({
            icon: 'check',
            tooltip: 'Approve request',
            onClick: (event, rowData) => this.onApprove(event, rowData),
            disabled: rowData.status !== "Pending"
          }),
          rowData => ({
            icon: 'close',
            tooltip: 'Deny request',
            onClick: (event, rowData) => this.onDeny(event, rowData),
            disabled: rowData.status !== "Pending"
          })
        ]}
        options={{
          actionsColumnIndex: -1,
          exportButton: true,
          grouping: true
        }}
      />
    )
  }
}

export default Table;