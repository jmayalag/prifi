package ch.epfl.prifiproxy.repository;

import android.app.Application;
import android.arch.lifecycle.LiveData;
import android.os.AsyncTask;

import java.util.List;

import ch.epfl.prifiproxy.persistence.AppDatabase;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationGroupDao;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class ConfigurationGroupRepository {
    private ConfigurationGroupDao groupDao;
    private LiveData<List<ConfigurationGroup>> allGroups;

    public ConfigurationGroupRepository(Application application) {
        AppDatabase db = AppDatabase.getDatabase(application);
        groupDao = db.configurationGroupDao();
        allGroups = groupDao.getAll();
    }

    public LiveData<List<ConfigurationGroup>> getAllGroups() {
        return allGroups;
    }

    public void insert(ConfigurationGroup group) {
        new InsertAsyncTask(groupDao).execute(group);
    }

    public void updateGroups(List<ConfigurationGroup> groups) {
        new UpdateAsyncTask(groupDao)
                .execute(groups.toArray(new ConfigurationGroup[groups.size()]));
    }

    private static class InsertAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationGroupDao dao;

        InsertAsyncTask(ConfigurationGroupDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(final ConfigurationGroup... configurationGroups) {
            dao.insert(configurationGroups);
            return null;
        }
    }

    private static class UpdateAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationGroupDao dao;

        UpdateAsyncTask(ConfigurationGroupDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(ConfigurationGroup... groups) {
            dao.update(groups);
            return null;
        }
    }

    private static class DeleteAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationGroupDao dao;

        DeleteAsyncTask(ConfigurationGroupDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(ConfigurationGroup... groups) {
            dao.delete(groups);
            return null;
        }
    }
}
