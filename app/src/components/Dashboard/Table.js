import React from "react";
import MaterialTable from "material-table";
import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import ListItemText from '@material-ui/core/ListItemText';
import NotesIcon from '@material-ui/icons/Notes';
import WcIcon from '@material-ui/icons/Wc';
import FaceIcon from '@material-ui/icons/Face';
import CommentIcon from '@material-ui/icons/Comment';
import moment from 'moment'
import RequestsService from '../../service/RequestsService';

class Table extends React.Component {
  onApprove = (event, request) => {
    let requestID = request._id
    event.preventDefault();
    RequestsService.approveRequestAdmin(requestID, this.props.config)
    .then(res => {
      if (res.status === 200) {
        this.props.handleChangeRequestStatus(requestID, "Approved")
      }})
    .catch(error => {
       alert("Internal server error")
    });
  }

  onDeny = (event, request) => {
    let requestID = request._id
    event.preventDefault();
    RequestsService.denyRequestAdmin(requestID, this.props.config)
    .then(res => {
      if (res.status === 200) {
        this.props.handleChangeRequestStatus(requestID, "Denied")
      }})
    .catch(error => {
       alert("Internal server error")
    });
  }

  render() {
    if(!this.props || this.props.config == null) {
      return <div>Loading...</div>
    }
    // Create a clone of props so that modifications here will not bubbled up
    let requests = JSON.parse(JSON.stringify(this.props.requests))
    // Transform timestamp and null value to be human-readable
    requests = requests.map(function(item){
      item.timestamp =  moment.parseZone(item.timestamp).local().format("MM/DD/YYYY HH:mm")
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
        title="All Requests"
        columns={[
          { title: 'ID', field: '_id' },
          { title: 'Username', field: 'username' },
          { title: 'Email', field: 'email' },
          { title: 'Application Submitted', field: 'timestamp', },
          { title: 'Status', field: 'status',},
          { title: 'Processed', field: 'processedTimestamp',},
          { title: 'Admin', field: 'admin',},
          { title: 'Assignees', field: 'assignees',},
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
                  <NotesIcon />
                </ListItemIcon>
                <ListItemText primary={rowData.info.applicationText}/>
              </ListItem>
              {/* Hide note section if the data does not contain it */}
              <ListItem button style={{display: rowData.note ? '' : 'none' }}>
                <ListItemIcon>
                  <CommentIcon />
                </ListItemIcon>
                <ListItemText primary={rowData.note}/>
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